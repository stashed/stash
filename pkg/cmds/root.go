package cmds

import (
	"flag"
	"log"
	"strings"

	v "github.com/appscode/go/version"
	"github.com/appscode/kutil/tools/analytics"
	"github.com/appscode/stash/client/scheme"
	"github.com/appscode/stash/pkg/util"
	"github.com/jpillora/go-ogle-analytics"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	gaTrackingCode = "UA-62096468-20"
)

func NewRootCmd() *cobra.Command {
	var (
		enableAnalytics = true
	)
	var rootCmd = &cobra.Command{
		Use:               "stash",
		Short:             `Stash by AppsCode - Backup your Kubernetes Volumes`,
		Long:              `Stash is a Kubernetes operator for restic. For more information, visit here: https://appscode.com/products/stash`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
			if enableAnalytics && gaTrackingCode != "" {
				if client, err := ga.NewClient(gaTrackingCode); err == nil {
					util.AnalyticsClientID = analytics.ClientID()
					client.ClientID(util.AnalyticsClientID)
					parts := strings.Split(c.CommandPath(), " ")
					client.Send(ga.NewEvent(parts[0], strings.Join(parts[1:], "/")).Label(v.Version.Version))
				}
			}
			scheme.AddToScheme(clientsetscheme.Scheme)
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})
	rootCmd.PersistentFlags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical events to Google Analytics")

	rootCmd.AddCommand(v.NewCmdVersion())
	rootCmd.AddCommand(NewCmdRun())
	rootCmd.AddCommand(NewCmdBackup())
	rootCmd.AddCommand(NewCmdRecover())
	rootCmd.AddCommand(NewCmdCheck())
	return rootCmd
}
