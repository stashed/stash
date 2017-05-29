package cmd

import (
	"github.com/appscode/log"
	_ "github.com/appscode/restik/api/install"
	"github.com/appscode/restik/pkg/controller"
	"github.com/spf13/cobra"
)

func NewCmdWatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Run restic backup",
		Run: func(cmd *cobra.Command, args []string) {
			cronController, err := controller.NewCronController()
			if err != nil {
				log.Fatalln(err)
			}
			err = cronController.RunBackup()
			if err != nil {
				log.Errorln(err)
			}
		},
	}
	return cmd
}
