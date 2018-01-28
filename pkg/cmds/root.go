package cmds

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/appscode/go/log/golog"
	v "github.com/appscode/go/version"
	"github.com/appscode/kutil/tools/analytics"
	"github.com/appscode/stash/client/scheme"
	"github.com/appscode/stash/pkg/admission/plugin"
	"github.com/appscode/stash/pkg/util"
	"github.com/jpillora/go-ogle-analytics"
	"github.com/openshift/generic-admission-server/pkg/cmd/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	gaTrackingCode = "UA-62096468-20"
)

func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               "stash",
		Short:             `Stash by AppsCode - Backup your Kubernetes Volumes`,
		Long:              `Stash is a Kubernetes operator for restic. For more information, visit here: https://appscode.com/products/stash`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
			if util.EnableAnalytics && gaTrackingCode != "" {
				if client, err := ga.NewClient(gaTrackingCode); err == nil {
					util.AnalyticsClientID = analytics.ClientID()
					client.ClientID(util.AnalyticsClientID)
					parts := strings.Split(c.CommandPath(), " ")
					client.Send(ga.NewEvent(parts[0], strings.Join(parts[1:], "/")).Label(v.Version.Version))
				}
			}
			scheme.AddToScheme(clientsetscheme.Scheme)
			util.LoggerOptions = golog.ParseFlags(c.Flags())
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})
	rootCmd.PersistentFlags().BoolVar(&util.EnableAnalytics, "analytics", util.EnableAnalytics, "Send analytical events to Google Analytics")

	rootCmd.AddCommand(v.NewCmdVersion())
	rootCmd.AddCommand(NewCmdRun())
	rootCmd.AddCommand(NewCmdBackup())
	rootCmd.AddCommand(NewCmdRecover())
	rootCmd.AddCommand(NewCmdCheck())

	stopCh := genericapiserver.SetupSignalHandler()
	cmd := server.NewCommandStartAdmissionServer(os.Stdout, os.Stderr, stopCh, &plugin.AdmissionHook{})
	cmd.Use = "admission-webhook"
	cmd.Long = "Launch Stash admission webhook server"
	cmd.Short = cmd.Long
	rootCmd.AddCommand(cmd)

	return rootCmd
}
