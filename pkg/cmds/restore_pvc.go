package cmds

import (
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/appscode/stash/pkg/restic"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/errors"
)

const (
	JobPVCRestore = "stash-pvc-restore"
)

func NewCmdRestorePVC() *cobra.Command {
	var (
		outputDir  string
		restoreOpt restic.RestoreOptions
		setupOpt   = restic.SetupOptions{
			ScratchDir:  restic.DefaultScratchDir,
			EnableCache: false,
		}
		metrics = restic.MetricsOptions{
			JobName: JobPVCRestore,
		}
	)

	cmd := &cobra.Command{
		Use:               "restore-pvc",
		Short:             "Takes a restore of Persistent Volume Claim",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "restore-dirs", "provider", "secret-dir")
			// init restic wrapper
			resticWrapper, err := restic.NewResticWrapper(setupOpt)
			if err != nil {
				return handleResticError(outputDir, restic.DefaultOutputFileName, err)
			}
			// Run restore
			restoreOutput, restoreErr := resticWrapper.RunRestore(restoreOpt)
			// If metrics are enabled then generate metrics
			if metrics.Enabled {
				err := restoreOutput.HandleMetrics(&metrics, restoreErr)
				if err != nil {
					return handleResticError(outputDir, restic.DefaultOutputFileName, errors.NewAggregate([]error{restoreErr, err}))
				}
			}
			if restoreErr != nil {
				return handleResticError(outputDir, restic.DefaultOutputFileName, restoreErr)
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
	cmd.Flags().StringVar(&setupOpt.Endpoint, "endpoint", setupOpt.Endpoint, "Endpoint for s3/s3 compatible backend")
	cmd.Flags().StringVar(&setupOpt.Path, "path", setupOpt.Path, "Directory inside the bucket where restore will be stored")
	cmd.Flags().StringVar(&setupOpt.SecretDir, "secret-dir", setupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().StringVar(&setupOpt.ScratchDir, "scratch-dir", setupOpt.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&setupOpt.EnableCache, "enable-cache", setupOpt.EnableCache, "Specify weather to enable caching for restic")

	cmd.Flags().StringVar(&restoreOpt.Host, "hostname", restoreOpt.Host, "Name of the host machine")
	cmd.Flags().StringSliceVar(&restoreOpt.RestoreDirs, "restore-dirs", restoreOpt.RestoreDirs, "List of directories to be restored")
	cmd.Flags().StringSliceVar(&restoreOpt.Snapshots, "snapshots", restoreOpt.Snapshots, "List of snapshots to be restored")

	cmd.Flags().StringVar(&outputDir, "output-dir", outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	cmd.Flags().BoolVar(&metrics.Enabled, "metrics-enabled", metrics.Enabled, "Specify weather to export Prometheus metrics")
	cmd.Flags().StringVar(&metrics.PushgatewayURL, "metrics-pushgateway-url", metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringVar(&metrics.MetricFileDir, "metrics-dir", metrics.MetricFileDir, "Directory where to write metric.prom file (keep empty if you don't want to write metric in a text file)")
	cmd.Flags().StringSliceVar(&metrics.Labels, "metrics-labels", metrics.Labels, "Labels to apply in exported metrics")

	return cmd
}
