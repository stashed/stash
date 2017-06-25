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
	"github.com/appscode/stash/pkg/cli"
	"github.com/appscode/stash/pkg/eventer"
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

type Options struct {
	App               string
	ResourceNamespace string
	ResourceName      string
	PrefixHostname    bool
	ScratchDir        string
	PushgatewayURL    string
	PodLabelsPath     string
}

type Scheduler struct {
	kubeClient      clientset.Interface
	stashClient     scs.ExtensionInterface
	opt             Options
	rchan           chan *sapi.Restic
	resourceVersion string
	locked          chan struct{}
	resticCLI       *cli.ResticWrapper
	cron            *cron.Cron
	recorder        record.EventRecorder
	syncPeriod      time.Duration
}

func New(kubeClient clientset.Interface, stashClient scs.ExtensionInterface, opt Options) *Scheduler {
	ctrl := &Scheduler{
		kubeClient:  kubeClient,
		stashClient: stashClient,
		opt:         opt,
		resticCLI:   cli.New(opt.ScratchDir, opt.PrefixHostname),
		rchan:       make(chan *sapi.Restic),
		recorder:    eventer.NewEventRecorder(kubeClient, "stash-scheduler"),
		syncPeriod:  30 * time.Second,
	}
	return ctrl
}

// Init and/or connect to repo
func (c *Scheduler) Setup() error {
	resource, err := c.stashClient.Restics(c.opt.ResourceNamespace).Get(c.opt.ResourceName)
	if err != nil {
		return err
	}
	if resource.Spec.Backend.RepositorySecretName == "" {
		return errors.New("Missing repository secret name")
	}
	secret, err := c.kubeClient.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.RepositorySecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	err = c.resticCLI.SetupEnv(resource, secret)
	if err != nil {
		return err
	}
	return c.resticCLI.InitRepositoryIfAbsent()
}

func (c *Scheduler) RunAndHold() {
	c.cron.Start()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.stashClient.Restics(c.opt.ResourceNamespace).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.stashClient.Restics(c.opt.ResourceNamespace).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&sapi.Restic{},
		c.syncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if r, ok := obj.(*sapi.Restic); ok {
					if r.Name == c.opt.ResourceName {
						c.rchan <- r
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
					c.rchan <- newObj
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

func (c *Scheduler) configureScheduler() error {
	r := <-c.rchan
	c.resourceVersion = r.ResourceVersion
	if c.cron == nil {
		c.locked = make(chan struct{})
		c.locked <- struct{}{}
		c.cron = cron.New()
	}

	if r.Spec.Backend.RepositorySecretName == "" {
		return errors.New("Missing repository secret name")
	}
	secret, err := c.kubeClient.CoreV1().Secrets(r.Namespace).Get(r.Spec.Backend.RepositorySecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	err = c.resticCLI.SetupEnv(r, secret)
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
		_, err = c.stashClient.Restics(r.Namespace).Update(r)
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

func (c *Scheduler) runOnce() (err error) {
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
	resource, err = c.stashClient.Restics(c.opt.ResourceNamespace).Get(c.opt.ResourceName)
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
	sessionID := strconv.FormatInt(startTime.Unix(), 10)
	var (
		restic_session_success = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   "restic",
			Subsystem:   "session",
			Name:        "success",
			Help:        "Indicates if session was successfully completed",
			ConstLabels: prometheus.Labels{"session": sessionID},
		})
		restic_session_fail = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   "restic",
			Subsystem:   "session",
			Name:        "fail",
			Help:        "Indicates if session failed",
			ConstLabels: prometheus.Labels{"session": sessionID},
		})
		restic_session_duration_seconds_total = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   "restic",
			Subsystem:   "session",
			Name:        "duration_seconds_total",
			Help:        "Total seconds taken to complete restic session",
			ConstLabels: prometheus.Labels{"session": sessionID},
		})
		restic_session_duration_seconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "restic",
			Subsystem:   "session",
			Name:        "duration_seconds",
			Help:        "Total seconds taken to complete restic session",
			ConstLabels: prometheus.Labels{"session": sessionID},
		}, []string{"session", "filegroup", "op"})
	)

	defer func() {
		endTime := metav1.Now()
		if err != nil {
			restic_session_success.Set(1)
			restic_session_fail.Set(0)
		} else {
			restic_session_success.Set(0)
			restic_session_fail.Set(1)
		}
		restic_session_duration_seconds_total.Set(endTime.Sub(startTime.Time).Seconds())

		if c.opt.PushgatewayURL != "" {
			push.Collectors(c.JobName(resource),
				c.GroupingKeys(resource),
				c.opt.PushgatewayURL,
				restic_session_success,
				restic_session_fail,
				restic_session_duration_seconds_total,
				restic_session_duration_seconds)
		}

		resource.Status.BackupCount++
		resource.Status.LastBackupTime = &startTime
		if resource.Status.FirstBackupTime == nil {
			resource.Status.FirstBackupTime = &startTime
		}
		resource.Status.LastBackupDuration = endTime.Sub(startTime.Time).String()
		if _, e2 := c.stashClient.Restics(resource.Namespace).Update(resource); e2 != nil {
			log.Errorf("Failed to update status for Restic %s@%s due to %s", resource.Name, resource.Namespace, err)
		}
	}()

	for _, fg := range resource.Spec.FileGroups {
		backupOpMetric := restic_session_duration_seconds.WithLabelValues(sessionID, sanitizeLabelValue(fg.Path), "backup")
		err = c.measure(c.resticCLI.Backup, resource, fg, backupOpMetric)
		if err != nil {
			log.Errorln("Backup operation failed for Reestic %s@%s due to %s", resource.Name, resource.Namespace, err)
			backupFailure()
			c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonFailedToBackup, " ERROR: "+err.Error())
			return
		} else {
			backupSuccess()
			c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonSuccessfulBackup, "Backup completed successfully.")
		}

		forgetOpMetric := restic_session_duration_seconds.WithLabelValues(sessionID, sanitizeLabelValue(fg.Path), "forget")
		err = c.measure(c.resticCLI.Forget, resource, fg, forgetOpMetric)
		if err != nil {
			log.Errorln("Failed to forget old snapshots for Restic %s@%s due to %s", resource.Name, resource.Namespace, err)
			c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonFailedToRetention, " ERROR: "+err.Error())
			return
		}
	}
	return
}

func (c *Scheduler) measure(f func(*sapi.Restic, sapi.FileGroup) error, resource *sapi.Restic, fg sapi.FileGroup, g prometheus.Gauge) (err error) {
	startTime := time.Now()
	defer func() {
		g.Set(time.Now().Sub(startTime).Seconds())
	}()
	err = f(resource, fg)
	return
}
