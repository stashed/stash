package backup

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	LeaderElectionLease = 3 * time.Second
)

func (c *Controller) BackupScheduler() error {
	stopBackup := make(chan struct{})
	defer close(stopBackup)

	// split code from here for leader election
	switch c.opt.Workload.Kind {
	case api.KindDeployment, api.KindReplicaSet, api.KindReplicationController:
		if err := c.electLeader(stopBackup); err != nil {
			return err
		}
	default:
		if err := c.setupAndRunScheduler(stopBackup); err != nil {
			return err
		}
	}
	select {} // no error, wait forever
}

func (c *Controller) setupAndRunScheduler(stopBackup chan struct{}) error {
	if _, repository, err := c.setup(); err != nil {
		err = fmt.Errorf("failed to setup backup. Error: %v", err)
		if repository != nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				BackupEventComponent,
				repository.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedSetup,
				err.Error(),
			)
		}
		return err
	}
	c.initResticWatcher() // setup restic watcher, not required for offline backup
	go c.runScheduler(stopBackup)
	return nil
}

func (c *Controller) electLeader(stopBackup chan struct{}) error {
	rlc := resourcelock.ResourceLockConfig{
		Identity:      c.opt.PodName,
		EventRecorder: c.recorder,
	}
	resLock, err := resourcelock.New(resourcelock.ConfigMapsResourceLock, c.opt.Namespace, util.GetConfigmapLockName(c.opt.Workload), c.k8sClient.CoreV1(), rlc)
	if err != nil {
		return fmt.Errorf("error during leader election: %s", err)
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
	return nil
}

func (c *Controller) runScheduler(stopCh chan struct{}) {
	c.cron.Start()
	c.locked <- struct{}{}

	c.stashInformerFactory.Start(stopCh)

	for _, v := range c.stashInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}

	c.rQueue.Run(stopCh)

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

	// check restic again, previously done in setup()
	restic, err := c.rLister.Restics(c.opt.Namespace).Get(c.opt.ResticName)
	if kerr.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	if restic.Spec.Backend.StorageSecretName == "" {
		return errors.New("missing repository secret name")
	}
	secret, err := c.k8sClient.CoreV1().Secrets(restic.Namespace).Get(restic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// setup restic again, previously done in setup()
	prefix := ""
	if prefix, err = c.resticCLI.SetupEnv(restic.Spec.Backend, secret, c.opt.SmartPrefix); err != nil {
		return err
	}
	if err = c.resticCLI.InitRepositoryIfAbsent(); err != nil {
		return err
	}
	repository, err := c.createRepositoryCrdIfNotExist(restic, prefix)
	if err != nil {
		return err
	}

	// run final restic backup command
	return c.runResticBackup(restic, repository)
}

func (c *Controller) checkOnceForScheduler() (err error) {

	var repository *api.Repository
	repository, err = c.stashClient.StashV1alpha1().Repositories(c.opt.Namespace).Get(c.opt.Workload.GetRepositoryCRDName(c.opt.PodName, c.opt.NodeName), metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		err = nil
		return
	} else if err != nil {
		return
	}

	select {
	case <-c.locked:
		log.Infof("Acquired lock for Repository %s/%s", repository.Namespace, repository.Name)
		defer func() {
			c.locked <- struct{}{}
		}()
	default:
		log.Warningf("Skipping checkup schedule for Repository %s/%s", repository.Namespace, repository.Name)
		return
	}

	err = c.resticCLI.Check()
	if err != nil {
		c.recorder.Eventf(
			repository.ObjectReference(),
			core.EventTypeWarning,
			eventer.EventReasonFailedToCheck,
			"Repository check failed for workload %s %s/%s. Reason: %v",
			c.opt.Workload.Kind, c.opt.Namespace, c.opt.Workload.Name, err)
	}
	return
}
