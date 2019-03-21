package cmds

import (
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/appscode/stash/pkg/restic"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/errors"
)

type pvcRestoreOptions struct {
	setupOpt   restic.SetupOptions
	restoreOpt restic.RestoreOptions
	outputDir  string
	metrics    restic.MetricsOptions
}

const (
	JobPVCRestore = "stash-pvc-restore"
)

func NewCmdRestorePVC() *cobra.Command {
	opt := pvcRestoreOptions{
		setupOpt: restic.SetupOptions{
			ScratchDir:  ScratchDir,
			EnableCache: false,
		},
		metrics: restic.MetricsOptions{
			JobName: JobPVCRestore,
		},
	}

	cmd := &cobra.Command{
		Use:               "restore-pvc",
		Short:             "Takes a restore of Persistent Volume Claim",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "restore-dirs", "provider", "secret-dir")
			// init restic wrapper
			resticWrapper, err := restic.NewResticWrapper(opt.setupOpt)
			if err != nil {
				return err
			}
			// Run restore
			restoreOutput, restoreErr := resticWrapper.RunRestore(opt.restoreOpt)
			// If metrics are enabled then generate metrics
			if opt.metrics.Enabled {
				err := restoreOutput.HandleMetrics(&opt.metrics, restoreErr)
				if err != nil {
					return errors.NewAggregate([]error{restoreErr, err})
				}
			}
			// If output directory specified, then write the output in "output.json" file in the specified directory
			if restoreErr == nil && opt.outputDir != "" {
				err := restoreOutput.WriteOutput(filepath.Join(opt.outputDir, OutputFileName))
				if err != nil {
					return err
				}
			}
			return restoreErr
		},
	}

	cmd.Flags().StringVar(&opt.setupOpt.Provider, "provider", opt.setupOpt.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&opt.setupOpt.Bucket, "bucket", opt.setupOpt.Bucket, "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&opt.setupOpt.Endpoint, "endpoint", opt.setupOpt.Endpoint, "Endpoint for s3/s3 compatible backend")
	cmd.Flags().StringVar(&opt.setupOpt.Path, "path", opt.setupOpt.Path, "Directory inside the bucket where restore will be stored")
	cmd.Flags().StringVar(&opt.setupOpt.SecretDir, "secret-dir", opt.setupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().StringVar(&opt.setupOpt.ScratchDir, "scratch-dir", opt.setupOpt.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&opt.setupOpt.EnableCache, "enable-cache", opt.setupOpt.EnableCache, "Specify weather to enable caching for restic")

	cmd.Flags().StringVar(&opt.restoreOpt.Host, "hostname", opt.restoreOpt.Host, "Name of the host machine")
	cmd.Flags().StringSliceVar(&opt.restoreOpt.RestoreDirs, "restore-dirs", opt.restoreOpt.RestoreDirs, "List of directories to be restored")
	cmd.Flags().StringSliceVar(&opt.restoreOpt.Snapshots, "snapshots", opt.restoreOpt.Snapshots, "List of snapshots to be restored")

	cmd.Flags().StringVar(&opt.outputDir, "output-dir", opt.outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	cmd.Flags().BoolVar(&opt.metrics.Enabled, "metrics-enabled", opt.metrics.Enabled, "Specify weather to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.metrics.PushgatewayURL, "metrics-pushgateway-url", opt.metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringVar(&opt.metrics.MetricFileDir, "metrics-dir", opt.metrics.MetricFileDir, "Directory where to write metric.prom file (keep empty if you don't want to write metric in a text file)")
	cmd.Flags().StringSliceVar(&opt.metrics.Labels, "metrics-labels", opt.metrics.Labels, "Labels to apply in exported metrics")

	return cmd
}
