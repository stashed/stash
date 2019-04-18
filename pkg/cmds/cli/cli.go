package cli

import (
	"path/filepath"

	cs "github.com/appscode/stash/client/clientset/versioned"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/homedir"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/clientcmd"
)

const (
	cliScratchDir = "/tmp/stash-cli/scratch"
	cliSecretDir  = "/tmp/stash-cli/secret"
)

type stashCLIController struct {
	clientConfig *rest.Config
	kubeClient   kubernetes.Interface
	stashClient  cs.Interface
}

func NewCLICmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "cli",
		Short:             `Stash CLI`,
		Long:              `Kubectl plugin for Stash`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.AddCommand(NewCopyRepositoryCmd())
	cmd.AddCommand(NewUnlockRepositoryCmd())
	cmd.AddCommand(NewUnlockLocalRepositoryCmd())
	cmd.AddCommand(NewTriggerBackupCmd())
	cmd.AddCommand(NewBackupPVCmd())
	cmd.AddCommand(NewDownloadCmd())

	return cmd
}

func newStashCLIController(kubeConfig string) (*stashCLIController, error) {
	var (
		controller = &stashCLIController{}
		err        error
	)
	if kubeConfig == "" && !meta_util.PossiblyInCluster() {
		kubeConfig = filepath.Join(homedir.HomeDir(), "/.kube/config")
	}
	if controller.clientConfig, err = clientcmd.BuildConfigFromContext(kubeConfig, ""); err != nil {
		return nil, err
	}
	if controller.kubeClient, err = kubernetes.NewForConfig(controller.clientConfig); err != nil {
		return nil, err
	}
	if controller.stashClient, err = cs.NewForConfig(controller.clientConfig); err != nil {
		return nil, err
	}
	return controller, nil
}
