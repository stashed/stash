package recovery

import (
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RecoveryOpt struct {
	Namespace    string
	RecoveryName string
	KubeClient   kubernetes.Interface
	StashClient  cs.StashV1alpha1Interface
}

func (opt *RecoveryOpt) RunRecovery() {
	recovery, err := opt.StashClient.Recoveries(opt.Namespace).Get(opt.RecoveryName, metav1.GetOptions{})
	if err != nil {
		log.Infoln(err)
		return
	}

	if err = recovery.IsValid(); err != nil {
		log.Infoln(err)
		opt.UpdateRecoveryStatus(recovery, "FAILED:"+err.Error())
		return
	}

	if err = opt.RecoverOrErr(recovery); err != nil {
		log.Infoln(err)
		opt.UpdateRecoveryStatus(recovery, "FAILED:"+err.Error())
		return
	}

	opt.UpdateRecoveryStatus(recovery, "SUCCEED")
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

	cli := cli.New("/tmp", "")
	if err = cli.SetupEnv(restic, secret, ""); err != nil {
		return err
	}

	for _, fg := range restic.Spec.FileGroups {
		if err = cli.Restore(fg.Path); err != nil {
			return err
		}
	}

	return nil
}

func (opt *RecoveryOpt) UpdateRecoveryStatus(recovery *api.Recovery, status string) {
	recovery.Status = status
	if _, err := opt.StashClient.Recoveries(opt.Namespace).Update(recovery); err != nil {
		log.Infoln("Error updating recovery status:", status)
	} else {
		log.Infoln("Updated recovery status:", status)
	}
}
