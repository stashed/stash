package main

import (
	"flag"
	"log"
	"os"

	"github.com/appscode/go/version"
	logs "github.com/appscode/log/golog"
	_ "github.com/appscode/restik/api/install"
	"github.com/appscode/restik/pkg/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	// Add fake package as a dependency to add this under vendor
	_ "github.com/appscode/restik/client/clientset/fake"
	_ "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	var rootCmd = &cobra.Command{
		Use: "restik",
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	rootCmd.AddCommand(version.NewCmdVersion())
	rootCmd.AddCommand(cmd.NewCmdRun(Version))
	rootCmd.AddCommand(cmd.NewCmdWatch(Version))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
