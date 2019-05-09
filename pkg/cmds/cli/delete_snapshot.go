package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"stash.appscode.dev/stash/pkg/cmds/docker"
	"stash.appscode.dev/stash/pkg/registry/snapshot"
	"stash.appscode.dev/stash/pkg/util"
)

func NewDeleteSnapshotCmd() *cobra.Command {
	var (
		kubeConfig string
		namespace  string
		localDirs  = &cliLocalDirectories{}
	)

	var cmd = &cobra.Command{
		Use:               "delete-snapshot",
		Short:             `Delete a snapshot from repository backend`,
		Long:              `Delete a snapshot from repository backend`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("snapshot name not provided")
			}
			repoName, snapshotId, err := util.GetRepoNameAndSnapshotID(args[0])
			if err != nil {
				return err
			}

			c, err := newStashCLIController(kubeConfig)
			if err != nil {
				return err
			}

			// get source repository
			repository, err := c.stashClient.StashV1alpha1().Repositories(namespace).Get(repoName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// delete from local backend
			if repository.Spec.Backend.Local != nil {
				r := snapshot.NewREST(c.clientConfig)
				return r.ForgetVersionedSnapshots(repository, []string{snapshotId}, false)
			}

			// get source repository secret
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
			defer os.RemoveAll(tempDir)

			// prepare local dirs
			if err = localDirs.prepareSecretDir(tempDir, secret); err != nil {
				return err
			}
			if err = localDirs.prepareConfigDir(tempDir, &setupOpt, nil); err != nil {
				return err
			}

			// run unlock inside docker
			if err = runDeleteSnapshotViaDocker(*localDirs, snapshotId); err != nil {
				return err
			}
			log.Infof("Snapshot %s deleted from repository %s/%s", snapshotId, namespace, repoName)
			return nil
		},
	}

	cmd.Flags().StringVar(&kubeConfig, "kubeconfig", kubeConfig, "Path of the Kube config file.")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "Namespace of the Repository.")

	cmd.Flags().StringVar(&image.Registry, "docker-registry", image.Registry, "Docker image registry")
	cmd.Flags().StringVar(&image.Tag, "image-tag", image.Tag, "Stash image tag")

	return cmd
}

func runDeleteSnapshotViaDocker(localDirs cliLocalDirectories, snapshotId string) error {
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
		image.ToContainerImage(),
		"docker",
		"delete-snapshot",
		"--snapshot", snapshotId,
	}
	log.Infoln("Running docker with args:", args)
	out, err := exec.Command("docker", args...).CombinedOutput()
	log.Infoln("Output:", string(out))
	return err
}
