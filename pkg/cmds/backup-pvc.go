package cmds

import (
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/appscode/stash/pkg/restic"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/errors"
)

type pvcBackupOptions struct {
	setupOpt  restic.SetupOptions
	backupOpt restic.BackupOptions
	outputDir string
	metrics   restic.MetricsOptions
}

const (
	OutputFileName = "output.json"
	JobPVCBackup   = "stash-pvc-backup"
	ScratchDir     = "/tmp/restic/scratch" // mount emptyDir volume in this path in function YAML
)

func NewCmdBackupPVC() *cobra.Command {
	opt := pvcBackupOptions{
		setupOpt: restic.SetupOptions{
			ScratchDir:  ScratchDir,
			EnableCache: false,
		},
		metrics: restic.MetricsOptions{
			JobName: JobPVCBackup,
		},
	}

	cmd := &cobra.Command{
		Use:               "backup-pvc",
		Short:             "Takes a backup of Persistent Volume Claim",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "backup-dirs", "provider", "secret-dir")

			// init restic wrapper
			resticWrapper, err := restic.NewResticWrapper(opt.setupOpt)
			if err != nil {
				return err
			}
			// Run backup
			backupOutput, backupErr := resticWrapper.RunBackup(&opt.backupOpt)
			// If metrics are enabled then generate metrics
			if opt.metrics.Enabled {
				err := backupOutput.HandleMetrics(&opt.metrics, backupErr)
				if err != nil {
					return errors.NewAggregate([]error{backupErr, err})
				}
			}
			// If output directory specified, then write the output in "output.json" file in the specified directory
			if backupErr == nil && opt.outputDir != "" {
				err := backupOutput.WriteOutput(filepath.Join(opt.outputDir, OutputFileName))
				if err != nil {
					return err
				}
			}
			return backupErr
		},
	}

	cmd.Flags().StringVar(&opt.setupOpt.Provider, "provider", opt.setupOpt.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&opt.setupOpt.Bucket, "bucket", opt.setupOpt.Bucket, "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&opt.setupOpt.Endpoint, "endpoint", opt.setupOpt.Endpoint, "Endpoint for s3/s3 compatible backend")
	cmd.Flags().StringVar(&opt.setupOpt.Path, "path", opt.setupOpt.Path, "Directory inside the bucket where backup will be stored")
	cmd.Flags().StringVar(&opt.setupOpt.SecretDir, "secret-dir", opt.setupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().StringVar(&opt.setupOpt.ScratchDir, "scratch-dir", opt.setupOpt.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&opt.setupOpt.EnableCache, "enable-cache", opt.setupOpt.EnableCache, "Specify weather to enable caching for restic")

	cmd.Flags().StringVar(&opt.backupOpt.Host, "hostname", opt.backupOpt.Host, "Name of the host machine")
	cmd.Flags().StringSliceVar(&opt.backupOpt.BackupDirs, "backup-dirs", opt.backupOpt.BackupDirs, "List of directories to be backed up")

	cmd.Flags().IntVar(&opt.backupOpt.RetentionPolicy.KeepLast, "retention-keep-last", opt.backupOpt.RetentionPolicy.KeepLast, "Specify value for retention strategy")
	cmd.Flags().IntVar(&opt.backupOpt.RetentionPolicy.KeepHourly, "retention-keep-hourly", opt.backupOpt.RetentionPolicy.KeepHourly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&opt.backupOpt.RetentionPolicy.KeepDaily, "retention-keep-daily", opt.backupOpt.RetentionPolicy.KeepDaily, "Specify value for retention strategy")
	cmd.Flags().IntVar(&opt.backupOpt.RetentionPolicy.KeepWeekly, "retention-keep-weekly", opt.backupOpt.RetentionPolicy.KeepWeekly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&opt.backupOpt.RetentionPolicy.KeepMonthly, "retention-keep-monthly", opt.backupOpt.RetentionPolicy.KeepMonthly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&opt.backupOpt.RetentionPolicy.KeepYearly, "retention-keep-yearly", opt.backupOpt.RetentionPolicy.KeepYearly, "Specify value for retention strategy")
	cmd.Flags().StringSliceVar(&opt.backupOpt.RetentionPolicy.KeepTags, "retention-keep-tags", opt.backupOpt.RetentionPolicy.KeepTags, "Specify value for retention strategy")
	cmd.Flags().BoolVar(&opt.backupOpt.RetentionPolicy.Prune, "retention-prune", opt.backupOpt.RetentionPolicy.Prune, "Specify weather to prune old snapshot data")
	cmd.Flags().BoolVar(&opt.backupOpt.RetentionPolicy.DryRun, "retention-dry-run", opt.backupOpt.RetentionPolicy.DryRun, "Specify weather to test retention policy without deleting actual data")

	cmd.Flags().StringVar(&opt.outputDir, "output-dir", opt.outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	cmd.Flags().BoolVar(&opt.metrics.Enabled, "metrics-enabled", opt.metrics.Enabled, "Specify weather to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.metrics.PushgatewayURL, "metrics-pushgateway-url", opt.metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringVar(&opt.metrics.MetricFileDir, "metrics-dir", opt.metrics.MetricFileDir, "Directory where to write metric.prom file (keep empty if you don't want to write metric in a text file)")
	cmd.Flags().StringSliceVar(&opt.metrics.Labels, "metrics-labels", opt.metrics.Labels, "Labels to apply in exported metrics")

	return cmd
}
