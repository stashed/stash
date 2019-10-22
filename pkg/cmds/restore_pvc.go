package cmds

import (
	"path/filepath"

	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
)

func NewCmdRestorePVC() *cobra.Command {
	var (
		outputDir  string
		restoreOpt = restic.RestoreOptions{
			Host: restic.DefaultHost,
		}
		setupOpt = restic.SetupOptions{
			ScratchDir:  restic.DefaultScratchDir,
			EnableCache: false,
		}
	)

	cmd := &cobra.Command{
		Use:               "restore-pvc",
		Short:             "Takes a restore of Persistent Volume Claim",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "restore-dirs", "provider", "secret-dir")

			var restoreOutput *restic.RestoreOutput
			restoreOutput, err := restorePVC(restoreOpt, setupOpt)
			if err != nil {
				restoreOutput = &restic.RestoreOutput{
					HostRestoreStats: []api_v1beta1.HostRestoreStats{
						{
							Hostname: restoreOpt.Host,
							Phase:    api_v1beta1.HostRestoreFailed,
							Error:    err.Error(),
						},
					},
				}
			}
			// If output directory specified, then write the output in "output.json" file in the specified directory
			if outputDir != "" {
				return restoreOutput.WriteOutput(filepath.Join(outputDir, restic.DefaultOutputFileName))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&setupOpt.Provider, "provider", setupOpt.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&setupOpt.Bucket, "bucket", setupOpt.Bucket, "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&setupOpt.Endpoint, "endpoint", setupOpt.Endpoint, "Endpoint for s3/s3 compatible backend or REST server URL")
	cmd.Flags().StringVar(&setupOpt.Path, "path", setupOpt.Path, "Directory inside the bucket where backed up data has been stored")
	cmd.Flags().StringVar(&setupOpt.SecretDir, "secret-dir", setupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().StringVar(&setupOpt.ScratchDir, "scratch-dir", setupOpt.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&setupOpt.EnableCache, "enable-cache", setupOpt.EnableCache, "Specify whether to enable caching for restic")
	cmd.Flags().IntVar(&setupOpt.MaxConnections, "max-connections", setupOpt.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")

	cmd.Flags().StringVar(&restoreOpt.Host, "hostname", restoreOpt.Host, "Name of the host machine")
	cmd.Flags().StringSliceVar(&restoreOpt.RestorePaths, "restore-paths", restoreOpt.RestorePaths, "List of paths to restore")
	cmd.Flags().StringSliceVar(&restoreOpt.Snapshots, "snapshots", restoreOpt.Snapshots, "List of snapshots to be restored")

	cmd.Flags().StringVar(&outputDir, "output-dir", outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	return cmd
}

func restorePVC(restoreOpt restic.RestoreOptions, setupOpt restic.SetupOptions) (*restic.RestoreOutput, error) {
	var err error
	// apply nice, ionice settings from env
	setupOpt.Nice, err = util.NiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	setupOpt.IONice, err = util.IONiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}

	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(setupOpt)
	if err != nil {
		return nil, err
	}
	// Run restore
	return resticWrapper.RunRestore(restoreOpt)
}
