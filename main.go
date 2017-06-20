package main

import (
	"flag"
	"log"
	"os"

	"github.com/appscode/go/version"
	logs "github.com/appscode/log/golog"
	_ "github.com/appscode/restik/api/install"
	_ "github.com/appscode/restik/client/clientset/fake"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	_ "k8s.io/client-go/kubernetes/fake"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	var rootCmd = &cobra.Command{
		Use:   "restik",
		Short: `Restik by AppsCode - Backup your Kubernetes Volumes`,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	rootCmd.AddCommand(version.NewCmdVersion())
	rootCmd.AddCommand(NewCmdRun(Version))
	rootCmd.AddCommand(NewCmdWatch(Version))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
