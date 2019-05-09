package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/homedir"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/clientcmd"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/cmds/docker"
	docker_image "stash.appscode.dev/stash/pkg/docker"
	"stash.appscode.dev/stash/pkg/restic"
)

const (
	secretDirName = "secret"
	configDirName = "config"
)

type stashCLIController struct {
	clientConfig *rest.Config
	kubeClient   kubernetes.Interface
	stashClient  cs.Interface
}

type cliLocalDirectories struct {
	secretDir   string // temp dir
	configDir   string // temp dir
	downloadDir string // user provided or, current working dir
}

var (
	image = docker_image.Docker{
		Registry: docker_image.ACRegistry,
		Image:    docker_image.ImageStash,
		Tag:      "latest", // TODO: update default release tag
	}
)

func NewCLICmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "cli",
		Short:             `Stash CLI`,
		Long:              `Kubectl plugin for Stash`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.AddCommand(NewCopyRepositoryCmd())
	cmd.AddCommand(NewUnlockRepositoryCmd())
	cmd.AddCommand(NewUnlockLocalRepositoryCmd())
	cmd.AddCommand(NewTriggerBackupCmd())
	cmd.AddCommand(NewBackupPVCmd())
	cmd.AddCommand(NewDownloadCmd())
	cmd.AddCommand(NewDeleteSnapshotCmd())

	return cmd
}

func newStashCLIController(kubeConfig string) (*stashCLIController, error) {
	var (
		controller = &stashCLIController{}
		err        error
	)
	if kubeConfig == "" && !meta_util.PossiblyInCluster() {
		kubeConfig = filepath.Join(homedir.HomeDir(), "/.kube/config")
	}
	if controller.clientConfig, err = clientcmd.BuildConfigFromContext(kubeConfig, ""); err != nil {
		return nil, err
	}
	if controller.kubeClient, err = kubernetes.NewForConfig(controller.clientConfig); err != nil {
		return nil, err
	}
	if controller.stashClient, err = cs.NewForConfig(controller.clientConfig); err != nil {
		return nil, err
	}
	return controller, nil
}

func (localDirs *cliLocalDirectories) prepareSecretDir(tempDir string, secret *core.Secret) error {
	// write repository secrets in a sub-dir insider tempDir
	localDirs.secretDir = filepath.Join(tempDir, secretDirName)
	if err := os.MkdirAll(localDirs.secretDir, 0755); err != nil {
		return err
	}
	for key, value := range secret.Data {
		if err := ioutil.WriteFile(filepath.Join(localDirs.secretDir, key), value, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (localDirs *cliLocalDirectories) prepareConfigDir(tempDir string, setupOpt *restic.SetupOptions, restoreOpt *restic.RestoreOptions) error {
	// write restic options in a sub-dir insider tempDir
	localDirs.configDir = filepath.Join(tempDir, configDirName)
	if err := os.MkdirAll(localDirs.secretDir, 0755); err != nil {
		return err
	}
	if setupOpt != nil {
		err := docker.WriteSetupOptionToFile(setupOpt, filepath.Join(localDirs.configDir, docker.SetupOptionsFile))
		if err != nil {
			return err
		}
	}
	if restoreOpt != nil {
		err := docker.WriteRestoreOptionToFile(restoreOpt, filepath.Join(localDirs.configDir, docker.RestoreOptionsFile))
		if err != nil {
			return err
		}
	}
	return nil
}

func (localDirs *cliLocalDirectories) prepareDownloadDir() (err error) {
	// if destination flag is not specified, restore in current directory
	if localDirs.downloadDir == "" {
		if localDirs.downloadDir, err = os.Getwd(); err != nil {
			return err
		}
	}
	return os.MkdirAll(localDirs.downloadDir, 0755)
}
