/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"flag"
	"os"

	"stash.appscode.dev/stash/client/clientset/versioned/scheme"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/log/golog"
	v "github.com/appscode/go/version"
	"github.com/spf13/cobra"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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

			utilruntime.Must(scheme.AddToScheme(clientsetscheme.Scheme))
			utilruntime.Must(scheme.AddToScheme(legacyscheme.Scheme))
			utilruntime.Must(ocscheme.AddToScheme(clientsetscheme.Scheme))
			utilruntime.Must(ocscheme.AddToScheme(legacyscheme.Scheme))
			cli.LoggerOptions = golog.ParseFlags(c.Flags())
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	logs.ParseFlags()
	rootCmd.PersistentFlags().StringVar(&util.ServiceName, "service-name", "stash-operator", "Stash service name.")
	rootCmd.PersistentFlags().BoolVar(&cli.EnableAnalytics, "enable-analytics", cli.EnableAnalytics, "Send analytical events to Google Analytics")

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

	rootCmd.AddCommand(NewCmdUpdateStatus())

	rootCmd.AddCommand(NewCmdCreateVolumeSnapshot())
	rootCmd.AddCommand(NewCmdRestoreVolumeSnapshot())

	rootCmd.AddCommand(NewCmdRunHook())

	return rootCmd
}
