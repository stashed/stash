package docker

import (
	"path/filepath"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	"stash.appscode.dev/stash/pkg/restic"
)

func NewDeleteSnapshotCmd() *cobra.Command {
	var snapshotID string
	var cmd = &cobra.Command{
		Use:               "delete-snapshot",
		Short:             `Delete a snapshot from repository backend`,
		Long:              `Delete a snapshot from repository backend`,
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
			// delete snapshots
			if _, err = resticWrapper.DeleteSnapshots([]string{snapshotID}); err != nil {
				return err
			}
			log.Infof("Delete completed")
			return nil
		},
	}
	cmd.Flags().StringVar(&snapshotID, "snapshot", snapshotID, "Snapshot ID to be deleted")
	return cmd
}
