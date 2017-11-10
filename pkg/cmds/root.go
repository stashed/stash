package cmds

import (
	"flag"
	"log"
	"strings"

	v "github.com/appscode/go/version"
	"github.com/appscode/stash/client/scheme"
	"github.com/jpillora/go-ogle-analytics"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	gaTrackingCode = "UA-62096468-20"
)

func NewCmdStash(version string) *cobra.Command {
	var (
		enableAnalytics = true
	)
	var rootCmd = &cobra.Command{
		Use:               "stash",
		Short:             `Stash by AppsCode - Backup your Kubernetes Volumes`,
		Long:              `Stash is a Kubernetes operator for restic. For more information, visit here: https://github.com/appscode/stash/tree/master/docs`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
			if enableAnalytics && gaTrackingCode != "" {
				if client, err := ga.NewClient(gaTrackingCode); err == nil {
					parts := strings.Split(c.CommandPath(), " ")
					client.Send(ga.NewEvent(parts[0], strings.Join(parts[1:], "/")).Label(version))
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
	rootCmd.AddCommand(NewCmdRun(version))
	rootCmd.AddCommand(NewCmdSchedule())
	rootCmd.AddCommand(NewCmdRecover())
	return rootCmd
}
