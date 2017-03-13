package cmd

import (
	_ "github.com/appscode/restik/api/install"
	"github.com/appscode/restik/pkg/controller"
	"github.com/spf13/cobra"
)

func NewCmdBackup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "",
		Run: func(cmd *cobra.Command, args []string) {
			controller.RunBackup()
		},
	}
	return cmd
}
