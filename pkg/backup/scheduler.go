package backup

import (
	"errors"
	"fmt"
	"time"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	LeaderElectionLease = 3 * time.Second
)

func (c *Controller) BackupScheduler() {
	stopBackup := make(chan struct{})
	defer close(stopBackup)

	// split code from here for leader election
	switch c.opt.Workload.Kind {
	case api.KindDeployment, api.KindReplicaSet, api.KindReplicationController:
		c.electLeader(stopBackup)
	default:
		c.setupAndRunScheduler(stopBackup)
	}

	// Wait forever
	select {}
}

func (c *Controller) setupAndRunScheduler(stopBackup chan struct{}) {
	if _, err := c.setup(); err != nil {
		log.Fatalf("Failed to setup backup: %s", err)
	}
	// setup restic watcher, not required for offline backup
	c.initResticWatcher()
	go c.runScheduler(1, stopBackup)
}

func (c *Controller) electLeader(stopBackup chan struct{}) {
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
					log.Infoln("Got leadership, preparing backup backup")
					c.setupAndRunScheduler(stopBackup)
				},
				OnStoppedLeading: func() {
					log.Infoln("Lost leadership, stopping backup backup")
					stopBackup <- struct{}{}
				},
			},
		})
	}()
}

func (c *Controller) runScheduler(threadiness int, stopCh chan struct{}) {
	c.cron.Start()
	c.locked <- struct{}{}

	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.rQueue.ShutDown()
	glog.Info("Starting Stash backup")

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
	glog.Info("Stopping Stash backup")
}

func (c *Controller) configureScheduler(r *api.Restic) error {
	// Remove previous jobs
	for _, v := range c.cron.Entries() {
		c.cron.Remove(v.ID)
	}
	_, err := c.cron.AddFunc(r.Spec.Schedule, func() {
		if err := c.runOnceForScheduler(); err != nil {
			c.recorder.Event(r.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedCronJob, err.Error())
			log.Errorln(err)
		}
	})
	if err != nil {
		return err
	}
	_, err = c.cron.AddFunc("0 0 */3 * *", func() { c.checkOnceForScheduler() })
	return err
}

func (c *Controller) runOnceForScheduler() error {
	select {
	case <-c.locked:
		log.Infof("Acquired lock for Restic %s/%s", c.opt.Namespace, c.opt.ResticName)
		defer func() {
			c.locked <- struct{}{}
		}()
	default:
		log.Warningf("Skipping backup schedule for Restic %s/%s", c.opt.Namespace, c.opt.ResticName)
		return nil
	}

	// check resource again, previously done in setup()
	resource, err := c.rLister.Restics(c.opt.Namespace).Get(c.opt.ResticName)
	if kerr.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	if resource.Spec.Backend.StorageSecretName == "" {
		return errors.New("missing repository secret name")
	}
	secret, err := c.k8sClient.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// setup restic again, previously done in setup()
	if err = c.resticCLI.SetupEnv(resource, secret, c.opt.SmartPrefix); err != nil {
		return err
	}
	if err = c.resticCLI.InitRepositoryIfAbsent(); err != nil {
		return err
	}

	// run final restic backup command
	return c.runResticBackup(resource)
}

func (c *Controller) checkOnceForScheduler() (err error) {
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
