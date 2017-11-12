package scheduler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	stash_util "github.com/appscode/stash/client/typed/stash/v1alpha1/util"
	stash_listers "github.com/appscode/stash/listers/stash/v1alpha1"
	"github.com/appscode/stash/pkg/cli"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/robfig/cron.v2"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	msec10              = 10 * 1000 * 1000 * time.Nanosecond
	maxAttempts         = 3
	LeaderElectionLease = 3 * time.Second
	ConfigMapPrefix     = "leader-lock-"
)

type Options struct {
	Workload         api.LocalTypedReference
	Namespace        string
	ResticName       string
	ScratchDir       string
	PushgatewayURL   string
	NodeName         string
	PodName          string
	SmartPrefix      string
	SnapshotHostname string
	PodLabelsPath    string
	ResyncPeriod     time.Duration
	MaxNumRequeues   int
}

type Controller struct {
	k8sClient   kubernetes.Interface
	stashClient cs.StashV1alpha1Interface
	opt         Options
	locked      chan struct{}
	resticCLI   *cli.ResticWrapper
	cron        *cron.Cron
	recorder    record.EventRecorder

	// Restic
	rQueue    workqueue.RateLimitingInterface
	rIndexer  cache.Indexer
	rInformer cache.Controller
	rLister   stash_listers.ResticLister
}

func New(k8sClient kubernetes.Interface, stashClient cs.StashV1alpha1Interface, opt Options) *Controller {
	return &Controller{
		k8sClient:   k8sClient,
		stashClient: stashClient,
		opt:         opt,
		cron:        cron.New(),
		locked:      make(chan struct{}, 1),
		resticCLI:   cli.New(opt.ScratchDir, opt.SnapshotHostname),
		recorder:    eventer.NewEventRecorder(k8sClient, "stash-scheduler"),
	}
}

// Init and/or connect to repo
func (c *Controller) Setup() error {
	resource, err := c.stashClient.Restics(c.opt.Namespace).Get(c.opt.ResticName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Infof("Found restic %s", resource.Name)
	if err := resource.IsValid(); err != nil {
		return err
	}
	secret, err := c.k8sClient.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Infof("Found repository secret %s", secret.Name)
	err = c.resticCLI.SetupEnv(resource, secret, c.opt.SmartPrefix)
	if err != nil {
		return err
	}
	// c.resticCLI.DumpEnv()
	// ignore error but helps debug bad setup.
	c.resticCLI.InitRepositoryIfAbsent()
	c.initResticWatcher()
	return nil
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	c.cron.Start()
	c.locked <- struct{}{}

	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.rQueue.ShutDown()
	glog.Info("Starting Stash scheduler")

	go c.rInformer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.rInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runResticWatcher, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Stopping Stash scheduler")
}

func (c *Controller) configureScheduler(r *api.Restic) error {
	// Remove previous jobs
	for _, v := range c.cron.Entries() {
		c.cron.Remove(v.ID)
	}
	_, err := c.cron.AddFunc(r.Spec.Schedule, func() {
		if err := c.runOnce(); err != nil {
			c.recorder.Event(r.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedCronJob, err.Error())
			log.Errorln(err)
		}
	})
	if err != nil {
		return err
	}
	_, err = c.cron.AddFunc("0 0 */3 * *", func() { c.checkOnce() })
	return err
}

func (c *Controller) checkOnce() (err error) {
	select {
	case <-c.locked:
		log.Infof("Acquired lock for Restic %s/%s", c.opt.Namespace, c.opt.ResticName)
		defer func() {
			c.locked <- struct{}{}
		}()
	default:
		log.Warningf("Skipping checkup schedule for Restic %s/%s", c.opt.Namespace, c.opt.ResticName)
		return
	}

	var resource *api.Restic
	resource, err = c.rLister.Restics(c.opt.Namespace).Get(c.opt.ResticName)
	if kerr.IsNotFound(err) {
		err = nil
		return
	} else if err != nil {
		return
	}

	err = c.resticCLI.Check()
	if err != nil {
		c.recorder.Eventf(resource.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToRecover, "Repository check failed for workload %s %s/%s. Reason: %v", c.opt.Workload.Kind, c.opt.Namespace, c.opt.Workload.Name, err)
	}
	return
}

func (c *Controller) runOnce() (err error) {
	select {
	case <-c.locked:
		log.Infof("Acquired lock for Restic %s/%s", c.opt.Namespace, c.opt.ResticName)
		defer func() {
			c.locked <- struct{}{}
		}()
	default:
		log.Warningf("Skipping backup schedule for Restic %s/%s", c.opt.Namespace, c.opt.ResticName)
		return
	}

	var resource *api.Restic
	resource, err = c.rLister.Restics(c.opt.Namespace).Get(c.opt.ResticName)
	if kerr.IsNotFound(err) {
		err = nil
		return
	} else if err != nil {
		return
	}

	if resource.Spec.Backend.StorageSecretName == "" {
		err = errors.New("missing repository secret name")
		return
	}
	var secret *core.Secret
	secret, err = c.k8sClient.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return
	}
	err = c.resticCLI.SetupEnv(resource, secret, c.opt.SmartPrefix)
	if err != nil {
		return err
	}
	// c.resticCLI.DumpEnv()

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
				restic_session_success.Set(0)
				restic_session_fail.Set(1)
			} else {
				restic_session_success.Set(1)
				restic_session_fail.Set(0)
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

		stash_util.PatchRestic(c.stashClient, resource, func(in *api.Restic) *api.Restic {
			in.Status.BackupCount++
			in.Status.LastBackupTime = &startTime
			if in.Status.FirstBackupTime == nil {
				in.Status.FirstBackupTime = &startTime
			}
			in.Status.LastBackupDuration = endTime.Sub(startTime.Time).String()
			return in
		})
	}()

	for _, fg := range resource.Spec.FileGroups {
		backupOpMetric := restic_session_duration_seconds.WithLabelValues(sanitizeLabelValue(fg.Path), "backup")
		err = c.measure(c.resticCLI.Backup, resource, fg, backupOpMetric)
		if err != nil {
			log.Errorln("Backup operation failed for Reestic %s/%s due to %s", resource.Namespace, resource.Name, err)
			c.recorder.Event(resource.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToBackup, " Error taking backup: "+err.Error())
			return
		} else {
			hostname, _ := os.Hostname()
			c.recorder.Event(resource.ObjectReference(), core.EventTypeNormal, eventer.EventReasonSuccessfulBackup, "Backed up pod:"+hostname+" path:"+fg.Path)
		}

		forgetOpMetric := restic_session_duration_seconds.WithLabelValues(sanitizeLabelValue(fg.Path), "forget")
		err = c.measure(c.resticCLI.Forget, resource, fg, forgetOpMetric)
		if err != nil {
			log.Errorln("Failed to forget old snapshots for Restic %s/%s due to %s", resource.Namespace, resource.Name, err)
			c.recorder.Event(resource.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToRetention, " Error forgetting snapshots: "+err.Error())
			return
		}
	}
	return
}

func (c *Controller) measure(f func(*api.Restic, api.FileGroup) error, resource *api.Restic, fg api.FileGroup, g prometheus.Gauge) (err error) {
	startTime := time.Now()
	defer func() {
		g.Set(time.Now().Sub(startTime).Seconds())
	}()
	err = f(resource, fg)
	return
}

func (c *Controller) SetupAndRun(stopBackup chan struct{}) {
	err := os.MkdirAll(c.opt.ScratchDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create scratch dir: %s", err)
	}
	err = ioutil.WriteFile(c.opt.ScratchDir+"/.stash", []byte("test"), 644)
	if err != nil {
		log.Fatalf("No write access in scratch dir: %s", err)
	}
	if err = c.Setup(); err != nil {
		log.Fatalf("Failed to setup scheduler: %s", err)
	}
	go c.Run(1, stopBackup)
}

func (c *Controller) ElectLeader(stopBackup chan struct{}) {
	rlc := resourcelock.ResourceLockConfig{
		Identity:      c.opt.PodName,
		EventRecorder: c.recorder,
	}
	resLock, err := resourcelock.New(resourcelock.ConfigMapsResourceLock, c.opt.Namespace, util.GetConfigmapLockName(c.opt.Workload), c.k8sClient.CoreV1(), rlc)
	if err != nil {
		log.Fatalln(err)
	}
	go func() {
		leaderelection.RunOrDie(leaderelection.LeaderElectionConfig{
			Lock:          resLock,
			LeaseDuration: LeaderElectionLease,
			RenewDeadline: LeaderElectionLease * 2 / 3,
			RetryPeriod:   LeaderElectionLease / 3,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(stop <-chan struct{}) {
					log.Infoln("Got leadership, preparing backup scheduler")
					c.SetupAndRun(stopBackup)
				},
				OnStoppedLeading: func() {
					log.Infoln("Lost leadership, stopping backup scheduler")
					stopBackup <- struct{}{}
				},
			},
		})
	}()
}
