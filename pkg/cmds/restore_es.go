package cmds

import (
	"fmt"
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/appscode/stash/pkg/restic"
	"github.com/codeskyblue/go-sh"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

func NewCmdRestoreES() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		namespace      string
		appBindingName string
		outputDir      string
		esArgs         string
		setupOpt       = restic.SetupOptions{
			ScratchDir:  restic.DefaultScratchDir,
			EnableCache: false,
		}
		restoreOpt = restic.RestoreOptions{
			RestoreDirs: []string{ESDataDir},
		}
		metrics = restic.MetricsOptions{
			JobName: JobESBackup,
		}
	)

	cmd := &cobra.Command{
		Use:               "restore-es",
		Short:             "Restores ElasticSearch DB Backup",
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
			// Run restore
			restoreOutput, restoreErr := resticWrapper.RunRestore(restoreOpt)

			// run separate shell to restore indices
			esShell := sh.NewSession()
			esShell.ShowCMD = true
			esShell.SetEnv("DB_SCHEME", appBinding.Spec.ClientConfig.Service.Scheme)
			esShell.SetEnv("DB_USER", string(appBindingSecret.Data[ESUser]))
			esShell.SetEnv("DB_PASSWORD", string(appBindingSecret.Data[ESPassword]))
			esShell.SetEnv("AUTH_PLUGIN", "SearchGuard") // TODO
			esShell.Command("es-tools.sh",
				"restore",
				fmt.Sprintf(`--host=%s`, appBinding.Spec.ClientConfig.Service.Name),
				fmt.Sprintf(`--data-dir=%s`, ESDataDir),
				"--", esArgs,
			)
			if err := esShell.Run(); err != nil {
				return err
			}

			// If metrics are enabled then generate metrics
			if metrics.Enabled {
				err := restoreOutput.HandleMetrics(&metrics, restoreErr)
				if err != nil {
					return errors.NewAggregate([]error{restoreErr, err})
				}
			}
			// If output directory specified, then write the output in "output.json" file in the specified directory
			if restoreErr == nil && outputDir != "" {
				err := restoreOutput.WriteOutput(filepath.Join(outputDir, restic.DefaultOutputFileName))
				if err != nil {
					return err
				}
			}
			return restoreErr
		},
	}

	cmd.Flags().StringVar(&esArgs, "es-args", esArgs, "Additional arguments")

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

	cmd.Flags().StringVar(&restoreOpt.Host, "hostname", restoreOpt.Host, "Name of the host machine")
	cmd.Flags().StringSliceVar(&restoreOpt.Snapshots, "snapshots", restoreOpt.Snapshots, "Snapshots to restore")

	cmd.Flags().StringVar(&outputDir, "output-dir", outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	cmd.Flags().BoolVar(&metrics.Enabled, "metrics-enabled", metrics.Enabled, "Specify weather to export Prometheus metrics")
	cmd.Flags().StringVar(&metrics.PushgatewayURL, "metrics-pushgateway-url", metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringVar(&metrics.MetricFileDir, "metrics-dir", metrics.MetricFileDir, "Directory where to write metric.prom file (keep empty if you don't want to write metric in a text file)")
	cmd.Flags().StringSliceVar(&metrics.Labels, "metrics-labels", metrics.Labels, "Labels to apply in exported metrics")

	return cmd
}
