package recovery

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	stash_util "github.com/appscode/stash/client/typed/stash/v1alpha1/util"
	"github.com/appscode/stash/pkg/cli"
	"github.com/appscode/stash/pkg/eventer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

type Controller struct {
	k8sClient    kubernetes.Interface
	stashClient  cs.StashV1alpha1Interface
	namespace    string
	recoveryName string
	recorder     record.EventRecorder
}

func New(k8sClient kubernetes.Interface, stashClient cs.StashV1alpha1Interface, namespace, name string) *Controller {
	return &Controller{
		k8sClient:    k8sClient,
		stashClient:  stashClient,
		namespace:    namespace,
		recoveryName: name,
		recorder:     eventer.NewEventRecorder(k8sClient, "stash-restorer"),
	}
}

func (c *Controller) Run() {
	recovery, err := c.stashClient.Recoveries(c.namespace).Get(c.recoveryName, metav1.GetOptions{})
	if err != nil {
		log.Errorln(err)
		return
	}

	if err = recovery.IsValid(); err != nil {
		log.Errorln(err)
		stash_util.SetRecoveryStatusPhase(c.stashClient, recovery, api.RecoveryFailed)
		c.recorder.Event(recovery.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToRecover, err.Error())
		return
	}

	if err = c.RecoverOrErr(recovery); err != nil {
		log.Errorln(err)
		stash_util.SetRecoveryStatusPhase(c.stashClient, recovery, api.RecoveryFailed)
		c.recorder.Event(recovery.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToRecover, err.Error())
		return
	}

	log.Infoln("Recovery succeed")
	stash_util.SetRecoveryStatusPhase(c.stashClient, recovery, api.RecoverySucceeded) // TODO: status.Stats
	c.recorder.Event(recovery.ObjectReference(), core.EventTypeNormal, eventer.EventReasonSuccessfulRecovery, "Recovery succeed")
}

func (c *Controller) RecoverOrErr(recovery *api.Recovery) error {
	restic, err := c.stashClient.Restics(c.namespace).Get(recovery.Spec.Restic, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if err = restic.IsValid(); err != nil {
		return err
	}
	if restic.Status.BackupCount < 1 {
		return fmt.Errorf("no backup found")
	}

	secret, err := c.k8sClient.CoreV1().Secrets(c.namespace).Get(restic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	nodeName := recovery.Spec.NodeName
	podName, _ := api.StatefulSetPodName(recovery.Spec.Workload.Name, recovery.Spec.PodOrdinal) // ignore error for other kinds
	hostname, smartPrefix, err := recovery.Spec.Workload.HostnamePrefix(podName, nodeName)
	if err != nil {
		return err
	}

	cli := cli.New("/tmp", hostname)
	if err = cli.SetupEnv(restic, secret, smartPrefix); err != nil {
		return err
	}

	var errRec error
	for _, fg := range restic.Spec.FileGroups {
		d, err := c.measure(cli.Restore, fg.Path, hostname)
		if err != nil {
			errRec = err
			c.recorder.Eventf(recovery.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToRecover, "failed to recover FileGroup %s. Reason: %v", fg.Path, err)
			stash_util.SetRecoveryStats(c.stashClient, recovery, fg.Path, d, api.RecoveryFailed)
		} else {
			stash_util.SetRecoveryStats(c.stashClient, recovery, fg.Path, d, api.RecoverySucceeded)
		}
	}

	return errRec
}

func (c *Controller) measure(f func(string, string) error, path, host string) (time.Duration, error) {
	startTime := time.Now()
	err := f(path, host)
	return time.Now().Sub(startTime), err
}
