package cmds

import (
	"github.com/appscode/go/log"
	cs "github.com/appscode/stash/client/clientset/versioned"
	"github.com/appscode/stash/pkg/backupsession"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
)

func NewBackupSession() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string

		opt = backupsession.Options{
			Namespace: meta.Namespace(),
		}
	)

	cmd := &cobra.Command{
		Use:               "backup-session",
		Short:             "create a BackupSession",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			stashClient := cs.NewForConfigOrDie(config)

			ctrl := backupsession.New(kubeClient, stashClient, opt)
			err = ctrl.CreateBackupSession()
			if err != nil {
				log.Fatal(err)
			}

		},
	}

	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.Name, "backupsession.name", "", "Set BackupSession Name")
	cmd.Flags().StringVar(&opt.Namespace, "backupsession.namespace", opt.Namespace, "Set BackupSession Namespace")

	return cmd
}
