package docker

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"stash.appscode.dev/stash/pkg/restic"
)

const (
	ScratchDir         = "/tmp/scratch"
	SecretDir          = "/tmp/secret"
	ConfigDir          = "/tmp/config"
	DestinationDir     = "/tmp/destination"
	SetupOptionsFile   = "setup.json"
	RestoreOptionsFile = "restore.json"
)

func NewDockerCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "docker",
		Short:             `Run restic commands inside Docker`,
		Long:              `Run restic commands inside Docker`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.AddCommand(NewUnlockRepositoryCmd())
	cmd.AddCommand(NewDownloadCmd())
	cmd.AddCommand(NewDeleteSnapshotCmd())

	return cmd
}

func WriteSetupOptionToFile(options *restic.SetupOptions, fileName string) error {
	jsonOutput, err := json.MarshalIndent(options, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(fileName, jsonOutput, 0755); err != nil {
		return err
	}
	return nil
}

func ReadSetupOptionFromFile(filename string) (*restic.SetupOptions, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	options := &restic.SetupOptions{}
	err = json.Unmarshal(data, options)
	if err != nil {
		return nil, err
	}

	return options, nil
}

func WriteRestoreOptionToFile(options *restic.RestoreOptions, fileName string) error {
	jsonOutput, err := json.MarshalIndent(options, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(fileName, jsonOutput, 0755); err != nil {
		return err
	}
	return nil
}

func ReadRestoreOptionFromFile(filename string) (*restic.RestoreOptions, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	options := &restic.RestoreOptions{}
	err = json.Unmarshal(data, options)
	if err != nil {
		return nil, err
	}

	return options, nil
}
