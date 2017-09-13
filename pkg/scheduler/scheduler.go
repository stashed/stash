package scheduler

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/appscode/log"
	sapi_v1alpha1 "github.com/appscode/stash/apis/stash/v1alpha1"
	scs "github.com/appscode/stash/client/typed/stash/v1alpha1"
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

const (
	msec10      = 10 * 1000 * 1000 * time.Nanosecond
	maxAttempts = 3
)

type Options struct {
	AppKind        string
	AppName        string
	Namespace      string
	ResticName     string
	ScratchDir     string
	PushgatewayURL string
	NodeName       string
	PodName        string
	SmartPrefix    string
	PodLabelsPath  string
	ResyncPeriod   time.Duration
}

func (opt Options) autoPrefix(resource *sapi_v1alpha1.Restic) string {
	switch resource.Spec.UseAutoPrefix {
	case sapi_v1alpha1.None:
		return ""
	case sapi_v1alpha1.NodeName:
		return opt.NodeName
	case sapi_v1alpha1.PodName:
		return opt.PodName
	default:
		return opt.SmartPrefix
	}
}

type Scheduler struct {
	kubeClient  clientset.Interface
	stashClient scs.ResticsGetter
	opt         Options
	rchan       chan *sapi_v1alpha1.Restic
	locked      chan struct{}
	resticCLI   *cli.ResticWrapper
	cron        *cron.Cron
	recorder    record.EventRecorder
}

func New(kubeClient clientset.Interface, stashClient scs.ResticsGetter, opt Options) *Scheduler {
	return &Scheduler{
		kubeClient:  kubeClient,
		stashClient: stashClient,
		opt:         opt,
		rchan:       make(chan *sapi_v1alpha1.Restic, 1),
		cron:        cron.New(),
		locked:      make(chan struct{}, 1),
		resticCLI:   cli.New(opt.ScratchDir),
		recorder:    eventer.NewEventRecorder(kubeClient, "stash-scheduler"),
	}
}

// Init and/or connect to repo
func (c *Scheduler) Setup() error {
	resource, err := c.stashClient.Restics(c.opt.Namespace).Get(c.opt.ResticName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Infof("Found restic %s", resource.Name)
	if resource.Spec.Backend.StorageSecretName == "" {
		return errors.New("Missing repository secret name")
	}
	secret, err := c.kubeClient.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Infof("Found repository secret %s", secret.Name)
	err = c.resticCLI.SetupEnv(resource, secret, c.opt.autoPrefix(resource))
	if err != nil {
		return err
	}
	c.resticCLI.DumpEnv()
	// ignore error but helps debug bad setup.
	c.resticCLI.InitRepositoryIfAbsent()
	return nil
}

func (c *Scheduler) RunAndHold() {
	c.cron.Start()
	c.locked <- struct{}{}

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.stashClient.Restics(c.opt.Namespace).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.stashClient.Restics(c.opt.Namespace).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&sapi_v1alpha1.Restic{},
		c.opt.ResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if r, ok := obj.(*sapi_v1alpha1.Restic); ok {
					if r.Name == c.opt.ResticName {
						c.rchan <- r
						err := c.configureScheduler()
						if err != nil {
							c.recorder.Eventf(
								r.ObjectReference(),
								apiv1.EventTypeWarning,
								eventer.EventReasonFailedToBackup,
								"Failed to start Stash scehduler reason %v", err,
							)
							log.Errorln(err)
						}
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldObj, ok := old.(*sapi_v1alpha1.Restic)
				if !ok {
					log.Errorln(errors.New("Invalid Restic object"))
					return
				}
				newObj, ok := new.(*sapi_v1alpha1.Restic)
				if !ok {
					log.Errorln(errors.New("Invalid Restic object"))
					return
				}
				if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) && newObj.Name == c.opt.ResticName {
					c.rchan <- newObj
					err := c.configureScheduler()
					if err != nil {
						c.recorder.Eventf(
							newObj.ObjectReference(),
							apiv1.EventTypeWarning,
							eventer.EventReasonFailedToBackup,
							"Failed to update Stash scheduler reason %v", err,
						)
						log.Errorln(err)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if r, ok := obj.(*sapi_v1alpha1.Restic); ok {
					if r.Name == c.opt.ResticName {
						c.cron.Stop()
					}
				}
			},
		})
	ctrl.Run(wait.NeverStop)
}

func (c *Scheduler) configureScheduler() error {
	r := <-c.rchan

	if r.Spec.Backend.StorageSecretName == "" {
		return errors.New("Missing repository secret name")
	}

	// Remove previous jobs
	for _, v := range c.cron.Entries() {
		c.cron.Remove(v.ID)
	}

	interval := r.Spec.Schedule
	if _, err := cron.Parse(interval); err != nil {
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
	_, err := c.cron.AddFunc(interval, func() {
		if err := c.runOnce(); err != nil {
			c.recorder.Event(r, apiv1.EventTypeWarning, eventer.EventReasonFailedCronJob, err.Error())
			log.Errorln(err)
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
		log.Infof("Acquired lock for Restic %s@%s", c.opt.ResticName, c.opt.Namespace)
		defer func() {
			c.locked <- struct{}{}
		}()
	default:
		log.Warningf("Skipping backup schedule for Restic %s@%s", c.opt.ResticName, c.opt.Namespace)
		return
	}

	var resource *sapi_v1alpha1.Restic
	resource, err = c.stashClient.Restics(c.opt.Namespace).Get(c.opt.ResticName, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		err = nil
		return
	} else if err != nil {
		return
	}

	if resource.Spec.Backend.StorageSecretName == "" {
		err = errors.New("Missing repository secret name")
		return
	}
	var secret *apiv1.Secret
	secret, err = c.kubeClient.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return
	}
	err = c.resticCLI.SetupEnv(resource, secret, c.opt.autoPrefix(resource))
	if err != nil {
		return err
	}
	c.resticCLI.DumpEnv()

	err = c.resticCLI.InitRepositoryIfAbsent()
	if err != nil {
		return err
	}

	startTime := metav1.Now()
	var (
		restic_session_success = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "success",
			Help:      "Indicates if session was successfully completed",
		})
		restic_session_fail = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "fail",
			Help:      "Indicates if session failed",
		})
		restic_session_duration_seconds_total = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "duration_seconds_total",
			Help:      "Total seconds taken to complete restic session",
		})
		restic_session_duration_seconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "restic",
			Subsystem: "session",
			Name:      "duration_seconds",
			Help:      "Total seconds taken to complete restic session",
		}, []string{"filegroup", "op"})
	)

	defer func() {
		endTime := metav1.Now()
		if c.opt.PushgatewayURL != "" {
			if err != nil {
				restic_session_success.Set(1)
				restic_session_fail.Set(0)
			} else {
				restic_session_success.Set(0)
				restic_session_fail.Set(1)
			}
			restic_session_duration_seconds_total.Set(endTime.Sub(startTime.Time).Seconds())

			push.Collectors(c.JobName(resource),
				c.GroupingKeys(resource),
				c.opt.PushgatewayURL,
				restic_session_success,
				restic_session_fail,
				restic_session_duration_seconds_total,
				restic_session_duration_seconds)
		}

		attempt := 0
		for ; attempt < maxAttempts; attempt = attempt + 1 {
			resource.Status.BackupCount++
			resource.Status.LastBackupTime = &startTime
			if resource.Status.FirstBackupTime == nil {
				resource.Status.FirstBackupTime = &startTime
			}
			resource.Status.LastBackupDuration = endTime.Sub(startTime.Time).String()
			_, err := c.stashClient.Restics(resource.Namespace).Update(resource)
			if err == nil {
				break
			}
			log.Errorf("Attempt %d failed to update status for Restic %s@%s due to %s.", attempt, resource.Name, resource.Namespace, err)
			time.Sleep(msec10)
			if kerr.IsConflict(err) {
				resource, err = c.stashClient.Restics(resource.Namespace).Get(resource.Name, metav1.GetOptions{})
				if err != nil {
					return
				}
			}
		}
		if attempt >= maxAttempts {
			err = fmt.Errorf("Failed to add sidecar for ReplicaSet %s@%s after %d attempts.", resource.Name, resource.Namespace, attempt)
			return
		}
	}()

	for _, fg := range resource.Spec.FileGroups {
		backupOpMetric := restic_session_duration_seconds.WithLabelValues(sanitizeLabelValue(fg.Path), "backup")
		err = c.measure(c.resticCLI.Backup, resource, fg, backupOpMetric)
		if err != nil {
			log.Errorln("Backup operation failed for Reestic %s@%s due to %s", resource.Name, resource.Namespace, err)
			c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonFailedToBackup, " Error taking backup: "+err.Error())
			return
		} else {
			hostname, _ := os.Hostname()
			c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonSuccessfulBackup, "Backed up pod:"+hostname+" path:"+fg.Path)
		}

		forgetOpMetric := restic_session_duration_seconds.WithLabelValues(sanitizeLabelValue(fg.Path), "forget")
		err = c.measure(c.resticCLI.Forget, resource, fg, forgetOpMetric)
		if err != nil {
			log.Errorln("Failed to forget old snapshots for Restic %s@%s due to %s", resource.Name, resource.Namespace, err)
			c.recorder.Event(resource, apiv1.EventTypeNormal, eventer.EventReasonFailedToRetention, " Error forgetting snapshots: "+err.Error())
			return
		}
	}
	return
}

func (c *Scheduler) measure(f func(*sapi_v1alpha1.Restic, sapi_v1alpha1.FileGroup) error, resource *sapi_v1alpha1.Restic, fg sapi_v1alpha1.FileGroup, g prometheus.Gauge) (err error) {
	startTime := time.Now()
	defer func() {
		g.Set(time.Now().Sub(startTime).Seconds())
	}()
	err = f(resource, fg)
	return
}
