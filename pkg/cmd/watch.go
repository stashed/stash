package cmd

import (
	_ "github.com/appscode/k8s-addons/api/install"
	"github.com/appscode/restik/pkg/controller"
	"github.com/spf13/cobra"
)

func NewCmdWatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Run restic backup",
		Run: func(cmd *cobra.Command, args []string) {
			controller.RunBackup()
		},
	}
	return cmd
}
