package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	"github.com/appscode/stash/pkg/cmds/docker"
	"github.com/appscode/stash/pkg/restic"
	"github.com/appscode/stash/pkg/util"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDownloadCmd() *cobra.Command {
	var (
		kubeConfig     string
		repositoryName string
		namespace      string
		localDirs      cliLocalDirectories
		restoreOpt     = restic.RestoreOptions{
			SourceHost:  restic.DefaultHost,
			Destination: docker.DestinationDir,
		}
	)

	var cmd = &cobra.Command{
		Use:               "download",
		Short:             `Download snapshots`,
		Long:              `Download contents of snapshots from Repository`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "repository")

			c, err := newStashCLIController(kubeConfig)
			if err != nil {
				return err
			}

			// get source repository
			repository, err := c.stashClient.StashV1alpha1().Repositories(namespace).Get(repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// unlock local backend
			if repository.Spec.Backend.Local != nil {
				return fmt.Errorf("can't restore from repository with local backend")
			}
			// get repository secret
			secret, err := c.kubeClient.CoreV1().Secrets(namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// configure restic wrapper
			extraOpt := util.ExtraOptions{
				SecretDir:   docker.SecretDir,
				EnableCache: false,
				ScratchDir:  docker.ScratchDir,
			}
			setupOpt, err := util.SetupOptionsForRepository(*repository, extraOpt)
			if err != nil {
				return fmt.Errorf("setup option for repository failed")
			}

			// write secret and config in a temp dir
			// cleanup whole tempDir dir at the end
			tempDir, err := ioutil.TempDir("", "stash-cli")
			if err != nil {
				return err
			}
			// cleanup whole tempDir dir at the end
			defer os.RemoveAll(tempDir)

			// prepare local dirs
			localDirs.secretDir = filepath.Join(tempDir, secretDirName)
			localDirs.configDir = filepath.Join(tempDir, configDirName)
			localDirs, err = prepareDockerVolumeForRestore(localDirs, *secret, setupOpt, restoreOpt)
			if err != nil {
				return err
			}

			// run restore inside docker
			if err = runRestoreViaDocker(localDirs); err != nil {
				return err
			}
			log.Infof("Repository %s/%s restored in path %s", namespace, repositoryName, restoreOpt.Destination)
			return nil
		},
	}

	cmd.Flags().StringVar(&kubeConfig, "kubeconfig", kubeConfig, "Path of the Kube config file.")
	cmd.Flags().StringVar(&repositoryName, "repository", repositoryName, "Name of the Repository.")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "Namespace of the Repository.")
	cmd.Flags().StringVar(&localDirs.downloadDir, "destination", localDirs.downloadDir, "Destination path where snapshot will be restored.")

	cmd.Flags().StringVar(&restoreOpt.SourceHost, "host", restoreOpt.SourceHost, "Name of the source host machine")
	cmd.Flags().StringSliceVar(&restoreOpt.RestoreDirs, "directories", restoreOpt.RestoreDirs, "List of directories to be restored")
	cmd.Flags().StringSliceVar(&restoreOpt.Snapshots, "snapshots", restoreOpt.Snapshots, "List of snapshots to be restored")

	cmd.Flags().StringVar(&image.Registry, "docker-registry", image.Registry, "Docker image registry for unlock job")
	cmd.Flags().StringVar(&image.Tag, "image-tag", image.Tag, "Stash image tag for unlock job")

	return cmd
}

func prepareDockerVolumeForRestore(localDirs cliLocalDirectories, secret core.Secret, setupOpt restic.SetupOptions, restoreOpt restic.RestoreOptions) (cliLocalDirectories, error) {
	// write repository secrets
	if err := os.MkdirAll(localDirs.secretDir, 0755); err != nil {
		return cliLocalDirectories{}, err
	}
	for key, value := range secret.Data {
		if err := ioutil.WriteFile(filepath.Join(localDirs.secretDir, key), value, 0755); err != nil {
			return cliLocalDirectories{}, err
		}
	}
	// write restic options
	err := docker.WriteSetupOptionToFile(&setupOpt, filepath.Join(localDirs.configDir, docker.SetupOptionsFile))
	if err != nil {
		return cliLocalDirectories{}, err
	}
	err = docker.WriteRestoreOptionToFile(&restoreOpt, filepath.Join(localDirs.configDir, docker.RestoreOptionsFile))
	if err != nil {
		return cliLocalDirectories{}, err
	}
	// if destination flag is not specified, restore in current directory
	if localDirs.downloadDir == "" {
		if localDirs.downloadDir, err = os.Getwd(); err != nil {
			return cliLocalDirectories{}, err
		}
	}
	// create local download dir
	if err := os.MkdirAll(localDirs.downloadDir, 0755); err != nil {
		return cliLocalDirectories{}, err
	}
	return localDirs, nil
}

func runRestoreViaDocker(localDirs cliLocalDirectories) error {
	// get current user
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	args := []string{
		"run",
		"--rm",
		"-u", currentUser.Uid,
		"-v", localDirs.configDir + ":" + docker.ConfigDir,
		"-v", localDirs.secretDir + ":" + docker.SecretDir,
		"-v", localDirs.downloadDir + ":" + docker.DestinationDir,
		image.ToContainerImage(),
		"docker",
		"download-snapshots",
	}
	log.Infoln("Running docker with args:", args)
	out, err := exec.Command("docker", args...).CombinedOutput()
	log.Infoln("Output:", string(out))
	return err
}
