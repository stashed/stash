package cmds

import (
	"fmt"
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/appscode/stash/pkg/restic"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

const (
	JobMongoBackup  = "stash-mongo-backup"
	MongoUser       = "username"
	MongoPassword   = "password"
	MongoDumpFile   = "dumpfile.sql"
	MongoDumpCMD    = "mongodump"
	MongoRestoreCMD = "mongorestore"
)

func NewCmdBackupMongo() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		namespace      string
		appBindingName string
		mongoArgs      string
		outputDir      string
		setupOpt       = restic.SetupOptions{
			ScratchDir:  restic.DefaultScratchDir,
			EnableCache: false,
		}
		backupOpt = restic.BackupOptions{
			StdinFileName: MongoDumpFile,
		}
		metrics = restic.MetricsOptions{
			JobName: JobMongoBackup,
		}
	)

	cmd := &cobra.Command{
		Use:               "backup-mongo",
		Short:             "Takes a backup of Mongo DB",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "app-binding", "provider", "secret-dir")

			// prepare client
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				return err
			}
			kubeClient, err := kubernetes.NewForConfig(config)
			if err != nil {
				return err
			}
			appCatalogClient, err := appcatalog_cs.NewForConfig(config)
			if err != nil {
				return err
			}

			// get app binding
			appBinding, err := appCatalogClient.AppcatalogV1alpha1().AppBindings(namespace).Get(appBindingName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// get secret
			appBindingSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(appBinding.Spec.Secret.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// init restic wrapper
			resticWrapper, err := restic.NewResticWrapper(setupOpt)
			if err != nil {
				return err
			}
			// hide password, don't print cmd
			resticWrapper.HideCMD()

			// setup pipe command
			backupOpt.StdinPipeCommand = restic.Command{
				Name: MongoDumpCMD,
				Args: []interface{}{
					"--host", appBinding.Spec.ClientConfig.Service.Name,
					"--port", fmt.Sprint(appBinding.Spec.ClientConfig.Service.Port),
					"--username", string(appBindingSecret.Data[MongoUser]),
					"--password=" + string(appBindingSecret.Data[MongoPassword]),
					"--archive",
				},
			}
			if mongoArgs != "" {
				backupOpt.StdinPipeCommand.Args = append(backupOpt.StdinPipeCommand.Args, mongoArgs)
			}

			// wait for DB ready
			waitForDBReady(appBinding.Spec.ClientConfig.Service.Name, appBinding.Spec.ClientConfig.Service.Port)

			// Run backup
			backupOutput, backupErr := resticWrapper.RunBackup(backupOpt)
			// If metrics are enabled then generate metrics
			if metrics.Enabled {
				err := backupOutput.HandleMetrics(&metrics, backupErr)
				if err != nil {
					return errors.NewAggregate([]error{backupErr, err})
				}
			}
			// If output directory specified, then write the output in "output.json" file in the specified directory
			if backupErr == nil && outputDir != "" {
				err := backupOutput.WriteOutput(filepath.Join(outputDir, restic.DefaultOutputFileName))
				if err != nil {
					return err
				}
			}
			return backupErr
		},
	}

	cmd.Flags().StringVar(&mongoArgs, "mongo-args", mongoArgs, "Additional arguments")

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "Namespace of Backup/Restore Session")
	cmd.Flags().StringVar(&appBindingName, "app-binding", appBindingName, "Name of the app binding")

	cmd.Flags().StringVar(&setupOpt.Provider, "provider", setupOpt.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&setupOpt.Bucket, "bucket", setupOpt.Bucket, "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&setupOpt.Endpoint, "endpoint", setupOpt.Endpoint, "Endpoint for s3/s3 compatible backend")
	cmd.Flags().StringVar(&setupOpt.Path, "path", setupOpt.Path, "Directory inside the bucket where backup will be stored")
	cmd.Flags().StringVar(&setupOpt.SecretDir, "secret-dir", setupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().StringVar(&setupOpt.ScratchDir, "scratch-dir", setupOpt.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&setupOpt.EnableCache, "enable-cache", setupOpt.EnableCache, "Specify weather to enable caching for restic")

	cmd.Flags().StringVar(&backupOpt.Host, "hostname", backupOpt.Host, "Name of the host machine")

	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepLast, "retention-keep-last", backupOpt.RetentionPolicy.KeepLast, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepHourly, "retention-keep-hourly", backupOpt.RetentionPolicy.KeepHourly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepDaily, "retention-keep-daily", backupOpt.RetentionPolicy.KeepDaily, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepWeekly, "retention-keep-weekly", backupOpt.RetentionPolicy.KeepWeekly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepMonthly, "retention-keep-monthly", backupOpt.RetentionPolicy.KeepMonthly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&backupOpt.RetentionPolicy.KeepYearly, "retention-keep-yearly", backupOpt.RetentionPolicy.KeepYearly, "Specify value for retention strategy")
	cmd.Flags().StringSliceVar(&backupOpt.RetentionPolicy.KeepTags, "retention-keep-tags", backupOpt.RetentionPolicy.KeepTags, "Specify value for retention strategy")
	cmd.Flags().BoolVar(&backupOpt.RetentionPolicy.Prune, "retention-prune", backupOpt.RetentionPolicy.Prune, "Specify weather to prune old snapshot data")
	cmd.Flags().BoolVar(&backupOpt.RetentionPolicy.DryRun, "retention-dry-run", backupOpt.RetentionPolicy.DryRun, "Specify weather to test retention policy without deleting actual data")

	cmd.Flags().StringVar(&outputDir, "output-dir", outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	cmd.Flags().BoolVar(&metrics.Enabled, "metrics-enabled", metrics.Enabled, "Specify weather to export Prometheus metrics")
	cmd.Flags().StringVar(&metrics.PushgatewayURL, "metrics-pushgateway-url", metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringVar(&metrics.MetricFileDir, "metrics-dir", metrics.MetricFileDir, "Directory where to write metric.prom file (keep empty if you don't want to write metric in a text file)")
	cmd.Flags().StringSliceVar(&metrics.Labels, "metrics-labels", metrics.Labels, "Labels to apply in exported metrics")

	return cmd
}
