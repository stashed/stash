/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"path/filepath"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/flags"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
	v1 "kmodules.xyz/offshoot-api/api/v1"
)

func NewCmdRestorePVC() *cobra.Command {
	opt := pvcOptions{
		restoreOpt: restic.RestoreOptions{
			Host: restic.DefaultHost,
		},
		setupOpt: restic.SetupOptions{
			ScratchDir:  restic.DefaultScratchDir,
			EnableCache: false,
		},
		masterURL:      "",
		kubeConfigPath: "",
		namespace:      meta.Namespace(),
	}

	cmd := &cobra.Command{
		Use:               "restore-pvc",
		Short:             "Takes a restore of Persistent Volume Claim",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "restore-dirs", "provider", "secret-dir")

			config, err := clientcmd.BuildConfigFromFlags(opt.masterURL, opt.kubeConfigPath)
			if err != nil {
				glog.Fatalf("Could not get Kubernetes config: %s", err)
				return err
			}
			opt.config = config
			opt.k8sClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)

			invoker, err := apis.ExtractRestoreInvokerInfo(opt.k8sClient, opt.stashClient, opt.invokerKind, opt.invokerName, opt.namespace)
			if err != nil {
				return err
			}

			for _, targetInfo := range invoker.TargetsInfo {
				if targetInfo.Target != nil && targetMatched(targetInfo.Target.Ref, opt.targetKind, opt.targetName) {

					opt.restoreOpt.Host, err = util.GetHostName(targetInfo.Target)
					if err != nil {
						return err
					}

					// run backup
					restoreOutput, err := opt.restorePVC(targetInfo.Target.Ref)
					if err != nil {
						restoreOutput = &restic.RestoreOutput{
							RestoreTargetStatus: api_v1beta1.RestoreMemberStatus{
								Ref: targetInfo.Target.Ref,
								Stats: []api_v1beta1.HostRestoreStats{
									{
										Hostname: opt.restoreOpt.Host,
										Phase:    api_v1beta1.HostRestoreFailed,
										Error:    err.Error(),
									},
								},
							},
						}
					}

					// If output directory specified, then write the output in "output.json" file in the specified directory
					if opt.outputDir != "" {
						return restoreOutput.WriteOutput(filepath.Join(opt.outputDir, restic.DefaultOutputFileName))
					}
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opt.masterURL, "master", opt.masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&opt.kubeConfigPath, "kubeconfig", opt.kubeConfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	cmd.Flags().StringVar(&opt.setupOpt.Provider, "provider", opt.setupOpt.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&opt.setupOpt.Bucket, "bucket", opt.setupOpt.Bucket, "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&opt.setupOpt.Endpoint, "endpoint", opt.setupOpt.Endpoint, "Endpoint for s3/s3 compatible backend or REST server URL")
	cmd.Flags().StringVar(&opt.setupOpt.Region, "region", opt.setupOpt.Region, "Region for s3/s3 compatible backend")
	cmd.Flags().StringVar(&opt.setupOpt.Path, "path", opt.setupOpt.Path, "Directory inside the bucket where backed up data will be stored")
	cmd.Flags().StringVar(&opt.setupOpt.SecretDir, "secret-dir", opt.setupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().StringVar(&opt.setupOpt.ScratchDir, "scratch-dir", opt.setupOpt.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&opt.setupOpt.EnableCache, "enable-cache", opt.setupOpt.EnableCache, "Specify whether to enable caching for restic")
	cmd.Flags().Int64Var(&opt.setupOpt.MaxConnections, "max-connections", opt.setupOpt.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")

	cmd.Flags().StringVar(&opt.restoreOpt.Host, "hostname", opt.restoreOpt.Host, "Name of the host machine")
	cmd.Flags().StringSliceVar(&opt.restoreOpt.RestorePaths, "restore-paths", opt.restoreOpt.RestorePaths, "List of paths to restore")
	cmd.Flags().StringSliceVar(&opt.restoreOpt.Exclude, "exclude", opt.restoreOpt.Exclude, "List of pattern for directory/file to ignore during restore. Stash will not restore those files that matches these patterns.")
	cmd.Flags().StringSliceVar(&opt.restoreOpt.Include, "include", opt.restoreOpt.Include, "List of pattern for directory/file to restore. Stash will restore only those files that matches these patterns.")
	cmd.Flags().StringSliceVar(&opt.restoreOpt.Snapshots, "snapshots", opt.restoreOpt.Snapshots, "List of snapshots to be restored")

	cmd.Flags().StringVar(&opt.invokerKind, "invoker-kind", opt.invokerKind, "Kind of the backup invoker")
	cmd.Flags().StringVar(&opt.invokerName, "invoker-name", opt.invokerName, "Name of the respective backup invoker")
	cmd.Flags().StringVar(&opt.targetName, "target-name", opt.targetName, "Name of the Target")
	cmd.Flags().StringVar(&opt.targetKind, "target-kind", opt.targetKind, "Kind of the Target")
	cmd.Flags().StringVar(&opt.outputDir, "output-dir", opt.outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	return cmd
}

func (opt *pvcOptions) restorePVC(targetRef api_v1beta1.TargetRef) (*restic.RestoreOutput, error) {
	var err error
	// apply nice, ionice settings from env
	opt.setupOpt.Nice, err = v1.NiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	opt.setupOpt.IONice, err = v1.IONiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}

	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(opt.setupOpt)
	if err != nil {
		return nil, err
	}
	// Run restore
	return resticWrapper.RunRestore(opt.restoreOpt, targetRef)
}
