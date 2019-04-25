package cmds

import (
	"flag"
	"os"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/log/golog"
	v "github.com/appscode/go/version"
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/client/clientset/versioned/scheme"
	stash_cli "github.com/appscode/stash/pkg/cmds/cli"
	"github.com/appscode/stash/pkg/cmds/docker"
	"github.com/appscode/stash/pkg/util"
	"github.com/spf13/cobra"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"kmodules.xyz/client-go/logs"
	"kmodules.xyz/client-go/tools/cli"
	ocscheme "kmodules.xyz/openshift/client/clientset/versioned/scheme"
)

func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               "stash",
		Short:             `Stash by AppsCode - Backup your Kubernetes Volumes`,
		Long:              `Stash is a Kubernetes operator for restic. For more information, visit here: https://appscode.com/products/stash`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			flags.DumpAll(c.Flags())
			cli.SendAnalytics(c, v.Version.Version)

			scheme.AddToScheme(clientsetscheme.Scheme)
			scheme.AddToScheme(legacyscheme.Scheme)
			ocscheme.AddToScheme(clientsetscheme.Scheme)
			ocscheme.AddToScheme(legacyscheme.Scheme)
			cli.LoggerOptions = golog.ParseFlags(c.Flags())
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	logs.ParseFlags()
	rootCmd.PersistentFlags().StringVar(&util.ServiceName, "service-name", "stash-operator", "Stash service name.")
	rootCmd.PersistentFlags().BoolVar(&cli.EnableAnalytics, "enable-analytics", cli.EnableAnalytics, "Send analytical events to Google Analytics")
	rootCmd.PersistentFlags().BoolVar(&apis.EnableStatusSubresource, "enable-status-subresource", apis.EnableStatusSubresource, "If true, uses sub resource for crds.")

	rootCmd.AddCommand(v.NewCmdVersion())
	stopCh := genericapiserver.SetupSignalHandler()
	rootCmd.AddCommand(NewCmdRun(os.Stdout, os.Stderr, stopCh))

	rootCmd.AddCommand(NewCmdBackup())
	rootCmd.AddCommand(NewCmdRecover())
	rootCmd.AddCommand(NewCmdCheck())
	rootCmd.AddCommand(NewCmdScaleDown())
	rootCmd.AddCommand(NewCmdSnapshots())
	rootCmd.AddCommand(NewCmdForget())
	rootCmd.AddCommand(NewCmdCreateBackupSession())
	rootCmd.AddCommand(NewCmdRestore())
	rootCmd.AddCommand(NewCmdRunBackup())

	rootCmd.AddCommand(NewCmdBackupPVC())
	rootCmd.AddCommand(NewCmdRestorePVC())

	rootCmd.AddCommand(NewCmdBackupPG())
	rootCmd.AddCommand(NewCmdRestorePG())

	rootCmd.AddCommand(NewCmdBackupMySql())
	rootCmd.AddCommand(NewCmdRestoreMySql())

	rootCmd.AddCommand(NewCmdBackupMongo())
	rootCmd.AddCommand(NewCmdRestoreMongo())

	rootCmd.AddCommand(NewCmdUpdateStatus())

	rootCmd.AddCommand(stash_cli.NewCLICmd())
	rootCmd.AddCommand(docker.NewDockerCmd())

	return rootCmd
}
