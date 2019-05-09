package docker

import (
	"path/filepath"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	"stash.appscode.dev/stash/pkg/restic"
)

func NewUnlockRepositoryCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "unlock-repository",
		Short:             `Unlock Restic Repository`,
		Long:              `Unlock Restic Repository`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setupOpt, err := ReadSetupOptionFromFile(filepath.Join(ConfigDir, SetupOptionsFile))
			if err != nil {
				return err
			}
			resticWrapper, err := restic.NewResticWrapper(*setupOpt)
			if err != nil {
				return err
			}
			// run unlock
			if err = resticWrapper.UnlockRepository(); err != nil {
				return err
			}
			log.Infof("Unlock completed")
			return nil
		},
	}
	return cmd
}
