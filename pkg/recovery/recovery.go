package recovery

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

type RecoveryOpt struct {
	Namespace    string
	RecoveryName string
	ResticName   string
	KubeClient   kubernetes.Interface
	StashClient  cs.StashV1alpha1Interface
	Recovery     *api.Recovery
	Restic       *api.Restic
	Secret       *apiv1.Secret
	resticCLI    *cli.ResticWrapper
}

func (opt *RecoveryOpt) RunRecovery() {
	var err error
	opt.Recovery, err = opt.StashClient.Recoveries(opt.Namespace).Get(opt.RecoveryName, metav1.GetOptions{})
	if err != nil {
		log.Infoln(err)
		return
	}

	if len(opt.Recovery.Spec.Restic) == 0 {
		log.Infoln("Restic name missing")
		opt.Recovery.Status.RecoveryStatus = "FAILED: Restic name missing"
		_, err = opt.StashClient.Recoveries(opt.Namespace).Update(opt.Recovery)
		if err != nil {
			log.Infoln("Recovery failed, error updating recovery status")
		}
		return
	}

	if len(opt.Recovery.Spec.VolumeMounts) == 0 {
		log.Infoln("Target volume not specified")
		opt.Recovery.Status.RecoveryStatus = "FAILED: Target volume not specified"
		_, err = opt.StashClient.Recoveries(opt.Namespace).Update(opt.Recovery)
		if err != nil {
			log.Infoln("Recovery failed, error updating recovery status")
		}
		return
	}

	opt.ResticName = opt.Recovery.Spec.Restic

	err = opt.RecoverOrErr()
	if err != nil {
		log.Infoln(err)
		opt.Recovery.Status.RecoveryStatus = "FAILED:" + err.Error()
		_, err = opt.StashClient.Recoveries(opt.Namespace).Update(opt.Recovery)
		if err != nil {
			log.Infoln("Recovery failed, error updating recovery status")
		}
		return
	}

	opt.Recovery.Status.RecoveryStatus = "SUCCEED"
	_, err = opt.StashClient.Recoveries(opt.Namespace).Update(opt.Recovery)
	if err != nil {
		log.Infoln("Recovery succeed, error updating recovery status")
	}

	log.Infoln("Recovery succeed, status updated")

	time.Sleep(time.Minute * 30)
}

func (opt *RecoveryOpt) RecoverOrErr() error {
	var err error
	if opt.Restic, err = opt.StashClient.Restics(opt.Namespace).Get(opt.ResticName, metav1.GetOptions{}); err != nil {
		log.Infoln(err)
		return err
	}

	if opt.Restic.Status.BackupCount < 1 {
		log.Infoln("No backup found")
		return fmt.Errorf("no backup found")
	}

	if len(opt.Restic.Spec.Backend.StorageSecretName) == 0 {
		log.Infoln("Secret name missing")
		return fmt.Errorf("secret name missing")
	}

	if opt.Secret, err = opt.KubeClient.CoreV1().Secrets(opt.Namespace).Get(opt.Restic.Spec.Backend.StorageSecretName, metav1.GetOptions{}); err != nil {
		log.Infoln(err)
		return err
	}

	cli := cli.New("/tmp", "")
	if err = cli.SetupEnv(opt.Restic, opt.Secret, ""); err != nil {
		log.Infoln(err)
		return err
	}

	if err = cli.Restore(opt.Recovery); err != nil {
		log.Infoln(err)
		return err
	}

	return nil
}
