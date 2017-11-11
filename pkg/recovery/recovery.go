package recovery

import (
	"fmt"

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

type RecoveryOpt struct {
	Namespace    string
	RecoveryName string
	KubeClient   kubernetes.Interface
	StashClient  cs.StashV1alpha1Interface
	Recorder     record.EventRecorder
}

func (opt *RecoveryOpt) RunRecovery() {
	recovery, err := opt.StashClient.Recoveries(opt.Namespace).Get(opt.RecoveryName, metav1.GetOptions{})
	if err != nil {
		log.Errorln(err)
		return
	}

	if err = recovery.IsValid(); err != nil {
		log.Errorln(err)
		stash_util.SetRecoveryStatusPhase(opt.StashClient, recovery, api.RecoveryFailed)
		opt.Recorder.Event(recovery.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedRecovery, err.Error())
		return
	}

	if err = opt.RecoverOrErr(recovery); err != nil {
		log.Errorln(err)
		stash_util.SetRecoveryStatusPhase(opt.StashClient, recovery, api.RecoveryFailed)
		opt.Recorder.Event(recovery.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedRecovery, err.Error())
		return
	}

	log.Infoln("Recovery succeed")
	stash_util.SetRecoveryStatusPhase(opt.StashClient, recovery, api.RecoverySucceeded) // TODO: status.Stats
	opt.Recorder.Event(recovery.ObjectReference(), core.EventTypeNormal, eventer.EventReasonSuccessfulRecovery, "Recovery succeed")
}

func (opt *RecoveryOpt) RecoverOrErr(recovery *api.Recovery) error {
	restic, err := opt.StashClient.Restics(opt.Namespace).Get(recovery.Spec.Restic, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if err = restic.IsValid(); err != nil {
		return err
	}
	if restic.Status.BackupCount < 1 {
		return fmt.Errorf("no backup found")
	}

	secret, err := opt.KubeClient.CoreV1().Secrets(opt.Namespace).Get(restic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	nodeName := recovery.Spec.NodeName
	podName, _ := api.StatefulSetPodName(recovery.Spec.Workload.Name, recovery.Spec.PodOrdinal) // ignore error for other kinds
	hostname, smartPrefix, err := recovery.Spec.Workload.HostnamePrefix(podName, nodeName)      // workload canonicalized during IsValid check
	if err != nil {
		return err
	}

	cli := cli.New("/tmp", hostname)
	if err = cli.SetupEnv(restic, secret, smartPrefix); err != nil {
		return err
	}

	for _, fg := range restic.Spec.FileGroups {
		if err = cli.Restore(fg.Path, hostname); err != nil {
			return err
		}
	}

	return nil
}
