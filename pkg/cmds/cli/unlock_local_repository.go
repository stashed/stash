package cli

import (
	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	"github.com/appscode/stash/pkg/restic"
	"github.com/spf13/cobra"
)

func NewUnlockLocalRepositoryCmd() *cobra.Command {
	var (
		setupOpt = restic.SetupOptions{
			Provider:    restic.ProviderLocal,
			ScratchDir:  restic.DefaultScratchDir,
			EnableCache: false,
		}
	)

	var cmd = &cobra.Command{
		Use:               "unlock-local-repository",
		Short:             `Unlock Restic Repository with Local Backend`,
		Long:              `Unlock Restic Repository with Local Backend`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "path", "secret-dir")

			resticWrapper, err := restic.NewResticWrapper(setupOpt)
			if err != nil {
				return err
			}
			if err = resticWrapper.UnlockRepository(); err != nil {
				return err
			}
			log.Info("Repository unlocked")
			return nil
		},
	}

	cmd.Flags().StringVar(&setupOpt.Path, "path", setupOpt.Path, "Directory inside the bucket where backup will be stored")
	cmd.Flags().StringVar(&setupOpt.SecretDir, "secret-dir", setupOpt.SecretDir, "Directory where storage secret has been mounted")

	return cmd
}
