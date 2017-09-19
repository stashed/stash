package cmds

import (
	"fmt"
	"net/http"
	"time"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/pat"
	sapi "github.com/appscode/stash/apis/stash"
	scs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/docker"
	"github.com/appscode/stash/pkg/migrator"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeClient  clientset.Interface
	stashClient scs.ResticsGetter

	scratchDir string = "/tmp"
)

func NewCmdRun(version string) *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		tag            string        = stringz.Val(version, "canary")
		address        string        = ":56790"
		resyncPeriod   time.Duration = 5 * time.Minute
	)

	cmd := &cobra.Command{
		Use:               "run",
		Short:             "Run Stash operator",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := docker.CheckDockerImageVersion(docker.ImageOperator, tag); err != nil {
				log.Fatalf(`Image %v:%v not found.`, docker.ImageOperator, tag)
			}

			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalln(err)
			}
			kubeClient = clientset.NewForConfigOrDie(config)
			stashClient = scs.NewForConfigOrDie(config)
			crdClient := apiextensionsclient.NewForConfigOrDie(config)

			ctrl := controller.New(kubeClient, crdClient, stashClient, tag, resyncPeriod)
			err = ctrl.Setup()
			if err != nil {
				log.Fatalln(err)
			}

			if err = migrator.NewMigrator(kubeClient, crdClient).RunMigration(); err != nil {
				log.Fatalln(err)
			}

			log.Infoln("Starting operator...")
			ctrl.Run()

			m := pat.New()
			m.Get("/metrics", promhttp.Handler())

			pattern := fmt.Sprintf("/%s/v1beta1/namespaces/%s/restics/%s/metrics", sapi.GroupName, PathParamNamespace, PathParamName)
			log.Infof("URL pattern: %s", pattern)
			m.Get(pattern, http.HandlerFunc(ExportSnapshots))

			http.Handle("/", m)
			log.Infoln("Listening on", address)
			log.Fatal(http.ListenAndServe(address, nil))
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&address, "address", address, "Address to listen on for web interface and telemetry.")
	cmd.Flags().StringVar(&scratchDir, "scratch-dir", scratchDir, "Directory used to store temporary files. Use an `emptyDir` in Kubernetes.")
	cmd.Flags().DurationVar(&resyncPeriod, "resync-period", resyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")

	return cmd
}
