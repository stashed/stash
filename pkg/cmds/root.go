package cmds

import (
	"flag"
	"log"

	v "github.com/appscode/go/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewCmdStash(version string) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "stash",
		Short: `Stash by AppsCode - Backup your Kubernetes Volumes`,
		Long:  `Stash is a Kubernetes operator for restic. For more information, visit here: https://github.com/appscode/stash/tree/master/docs`,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	rootCmd.AddCommand(v.NewCmdVersion())
	rootCmd.AddCommand(NewCmdRun(version))
	rootCmd.AddCommand(NewCmdSchedule(version))
	return rootCmd
}
