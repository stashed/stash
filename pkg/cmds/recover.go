package cmds

import (
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil/meta"
	cs "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/recovery"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdRecover() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		recoveryName   string
		backoffMaxWait time.Duration
	)

	cmd := &cobra.Command{
		Use:               "recover",
		Short:             "Recover restic backup",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalln(err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			stashClient := cs.NewForConfigOrDie(config)

			c := recovery.New(kubeClient, stashClient, meta.Namespace(), recoveryName, backoffMaxWait)
			c.Run()
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&recoveryName, "recovery-name", recoveryName, "Name of the Recovery CRD.")
	cmd.Flags().DurationVar(&backoffMaxWait, "backoff-max-wait", 0, "Maximum wait for initial response from kube apiserver; 0 disables the timeout")

	return cmd
}
