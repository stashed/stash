package scheduler

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
	scs "github.com/appscode/stash/client/clientset"
	"github.com/appscode/stash/pkg/eventer"
	shell "github.com/codeskyblue/go-sh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/robfig/cron.v2"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

const (
	resticExe = "/restic"
)

type Options struct {
	Workload          string
	ResourceNamespace string
	ResourceName      string

	PrefixHostname bool
	ScratchDir     string
	PushgatewayURL string
	PodLabelsPath  string
}

type controller struct {
	KubeClient  clientset.Interface
	StashClient scs.ExtensionInterface

	opt             Options
	resource        chan *sapi.Restic
	resourceVersion string
	locked          chan struct{}

	sh       *shell.Session
	cron     *cron.Cron
	recorder record.EventRecorder
}

func NewController(kubeClient clientset.Interface, stashClient scs.ExtensionInterface, opt Options) *controller {
	ctrl := &controller{
		KubeClient:  kubeClient,
		StashClient: stashClient,
		opt:         opt,
		sh:          shell.NewSession(),
		resource:    make(chan *sapi.Restic),
		recorder:    eventer.NewEventRecorder(kubeClient, "stash-scheduler"),
	}
	ctrl.sh.SetDir(ctrl.opt.ScratchDir)
	ctrl.sh.ShowCMD = true
	return ctrl
}

// Init and/or connect to repo
func (c *controller) Setup() error {
	resource, err := c.StashClient.Restics(c.opt.ResourceNamespace).Get(c.opt.ResourceName)
	if err != nil {
		return err
	}

	err = c.SetEnvVars(resource)
	if err != nil {
		return err
	}

	if err = c.sh.Command(resticExe, "snapshots", "--json").Run(); err != nil {
		err = c.sh.Command(resticExe, "init").Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *controller) RunAndHold() {
	c.cron.Start()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.StashClient.Restics(c.opt.ResourceNamespace).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.StashClient.Restics(c.opt.ResourceNamespace).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&sapi.Restic{},
		time.Minute*2,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if r, ok := obj.(*sapi.Restic); ok {
					if r.Name == c.opt.ResourceName {
						c.resource <- r
						err := c.configureScheduler()
						if err != nil {
							schedulerFailedToAdd()
							c.recorder.Eventf(
								r,
								apiv1.EventTypeWarning,
								eventer.EventReasonFailedToBackup,
								"Failed to start backup process reason %v", err,
							)
							log.Errorln(err)
						} else {
							schedulerSuccessfullyAdded()
						}
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*sapi.Restic)
				if !ok {
					log.Errorln(errors.New("Error validating Stash object"))
					return
				}
				newObj, ok := new.(*sapi.Restic)
				if !ok {
					log.Errorln(errors.New("Error validating Stash object"))
					return
				}
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) && newObj.Name == c.opt.ResourceName {
					c.resource <- newObj
					err := c.configureScheduler()
					if err != nil {
						schedulerFailedToModify()
						c.recorder.Eventf(
							newObj,
							apiv1.EventTypeWarning,
							eventer.EventReasonFailedToBackup,
							"Failed to update backup process reason %v", err,
						)
						log.Errorln(err)
					} else {
						schedulerSuccessfullyModified()
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if r, ok := obj.(*sapi.Restic); ok {
					if r.Name == c.opt.ResourceName {
						c.cron.Stop()
					}
				}
			},
		})
	ctrl.Run(wait.NeverStop)
}

func (c *controller) configureScheduler() error {
	r := <-c.resource
	c.resourceVersion = r.ResourceVersion
	if c.cron == nil {
		c.locked = make(chan struct{})
		c.locked <- struct{}{}
		c.cron = cron.New()
	}

	err := c.SetEnvVars(r)
	if err != nil {
		return err
	}

	// Remove previous jobs
	for _, v := range c.cron.Entries() {
		c.cron.Remove(v.ID)
	}

	interval := r.Spec.Schedule
	if _, err = cron.Parse(interval); err != nil {
		log.Errorln(err)
		c.recorder.Event(r, apiv1.EventTypeWarning, eventer.EventReasonInvalidCronExpression, err.Error())
		//Reset Wrong Schedule
		r.Spec.Schedule = ""
		_, err = c.StashClient.Restics(r.Namespace).Update(r)
		if err != nil {
			return err
		}
		c.recorder.Event(r, apiv1.EventTypeNormal, eventer.EventReasonSuccessfulCronExpressionReset, "Cron expression reset")
		return nil
	}
	_, err = c.cron.AddFunc(interval, func() {
		if err := c.runOnce(); err != nil {
			stashJobFailure()
			c.recorder.Event(r, apiv1.EventTypeWarning, eventer.EventReasonFailedCronJob, err.Error())
			log.Errorln(err)
		} else {
			stashJobSuccess()
		}
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) runOnce() (err error) {
	select {
	case <-c.locked:
		log.Infof("Acquired lock for Restic %s@%s", c.opt.ResourceName, c.opt.ResourceNamespace)
		defer func() {
			c.locked <- struct{}{}
		}()
	default:
		log.Warningf("Skipping backup schedule for Restic %s@%s", c.opt.ResourceName, c.opt.ResourceNamespace)
		return
	}

	var resource *sapi.Restic
	resource, err = c.StashClient.Restics(c.opt.ResourceNamespace).Get(c.opt.ResourceName)
	if kerr.IsNotFound(err) {
		err = nil
		return
	} else if err != nil {
		return
	}
	if resource.ResourceVersion != c.resourceVersion {
		return fmt.Errorf("Restic %s@%s version %s does not match expected version %s", resource.Name, resource.Namespace, resource.ResourceVersion, c.resourceVersion)
	}

	startTime := metav1.Now()
	defer func() {
		endTime := metav1.Now()
		totalRuns := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "testname1",
			Help: "testhelp1",
		})
		totalRuns.Inc()
		totalFails := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "testname1",
			Help: "testhelp1",
		})
		totalDuration := prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "testname2",
			Help:        "testhelp2",
			ConstLabels: prometheus.Labels{"foo": "bar", "dings": "bums"},
		})
		totalDuration.Set(endTime.Sub(startTime.Time).Seconds())
		job := resource.Namespace + "-" + c.opt.Workload
		push.Collectors(job, c.GroupingKeys(resource), c.opt.PushgatewayURL, totalRuns, totalFails, totalDuration)

		resource.Status.BackupCount++
		resource.Status.LastBackupTime = &startTime
		if resource.Status.FirstBackupTime == nil {
			resource.Status.FirstBackupTime = &startTime
		}
		resource.Status.LastBackupDuration = endTime.Sub(startTime.Time).String()
		if _, e2 := c.StashClient.Restics(resource.Namespace).Update(resource); e2 != nil {
			log.Errorf("Failed to update status for Restic %s@%s due to %s", resource.Name, resource.Namespace, err)
		}
	}()
	for _, fg := range resource.Spec.FileGroups {
		err = c.processFileGroup(resource, fg, strconv.FormatInt(startTime.Unix(), 10))
		if err != nil {
			return
		}
	}
	return
}

func (c *controller) processFileGroup(resource *sapi.Restic, fg sapi.FileGroup, sessionID string) (err error) {
	startTime := time.Now()
	defer func() {
		d := time.Now().Sub(startTime)
		if c.opt.PushgatewayURL != "" {
			metric2 := prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "restic_backup_duration_seconds",
				Help:        "Seconds taken to backup a filegroup",
				ConstLabels: prometheus.Labels{"filegroup": sanitizeLabelValue(fg.Path), "session": sessionID},
			})
			metric2.Set(d.Seconds())

			job := resource.Namespace + "-" + c.opt.Workload
			push.Collectors(job, c.GroupingKeys(resource), c.opt.PushgatewayURL, metric2)
		}
	}()

	err = c.takeBackup(resource, fg)
	if err != nil {
		log.Errorln("Backup operation failed for Reestic %s@%s due to %s", resource.Name, resource.Namespace, err)
		backupFailure()
		c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonFailedToBackup, " ERROR: "+err.Error())
	} else {
		backupSuccess()
		c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonSuccessfulBackup, "Backup completed successfully.")
	}

	err = c.forgetSnapshots(resource, fg)
	if err != nil {
		log.Errorln("Failed to forget old snapshots for Restic %s@%s due to %s", resource.Name, resource.Namespace, err)
		c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonFailedToRetention, " ERROR: "+err.Error())
	}

	return
}

func (c *controller) takeBackup(resource *sapi.Restic, fg sapi.FileGroup) error {
	args := []interface{}{"backup", fg.Path, "--force"}
	// add tags if any
	for _, tag := range fg.Tags {
		args = append(args, "--tag")
		args = append(args, tag)
	}
	return c.sh.Command(resticExe, args...).Run()
}

func (c *controller) forgetSnapshots(resource *sapi.Restic, fg sapi.FileGroup) error {
	args := []interface{}{"forget"}
	if fg.RetentionPolicy.KeepLastSnapshots > 0 {
		args = append(args, sapi.KeepLast)
		args = append(args, fg.RetentionPolicy.KeepLastSnapshots)
	}
	if fg.RetentionPolicy.KeepHourlySnapshots > 0 {
		args = append(args, sapi.KeepHourly)
		args = append(args, fg.RetentionPolicy.KeepHourlySnapshots)
	}
	if fg.RetentionPolicy.KeepDailySnapshots > 0 {
		args = append(args, sapi.KeepDaily)
		args = append(args, fg.RetentionPolicy.KeepDailySnapshots)
	}
	if fg.RetentionPolicy.KeepWeeklySnapshots > 0 {
		args = append(args, sapi.KeepWeekly)
		args = append(args, fg.RetentionPolicy.KeepWeeklySnapshots)
	}
	if fg.RetentionPolicy.KeepMonthlySnapshots > 0 {
		args = append(args, sapi.KeepMonthly)
		args = append(args, fg.RetentionPolicy.KeepMonthlySnapshots)
	}
	if fg.RetentionPolicy.KeepYearlySnapshots > 0 {
		args = append(args, sapi.KeepYearly)
		args = append(args, fg.RetentionPolicy.KeepYearlySnapshots)
	}
	for _, tag := range fg.RetentionPolicy.KeepTags {
		args = append(args, "--keep-tag")
		args = append(args, tag)
	}
	for _, tag := range fg.Tags {
		args = append(args, "--tag")
		args = append(args, tag)
	}
	err := c.sh.Command(resticExe, args...).Run()
	if err != nil {
		return err
	}
	return nil
}
