/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"os"

	"stash.appscode.dev/apimachinery/client/clientset/versioned/scheme"

	"github.com/spf13/cobra"
	v "gomodules.xyz/x/version"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	ocscheme "kmodules.xyz/openshift/client/clientset/versioned/scheme"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "stash",
		Short:             `Stash by AppsCode - Backup your Kubernetes Volumes`,
		Long:              `Stash is a Kubernetes operator for restic. For more information, visit here: https://appscode.com/products/stash`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			utilruntime.Must(scheme.AddToScheme(clientsetscheme.Scheme))
			utilruntime.Must(scheme.AddToScheme(legacyscheme.Scheme))
			utilruntime.Must(ocscheme.AddToScheme(clientsetscheme.Scheme))
			utilruntime.Must(ocscheme.AddToScheme(legacyscheme.Scheme))
		},
	}

	rootCmd.AddCommand(v.NewCmdVersion())
	stopCh := genericapiserver.SetupSignalHandler()
	rootCmd.AddCommand(NewCmdRun(os.Stdout, os.Stderr, stopCh))

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
