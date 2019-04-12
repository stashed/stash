package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	"github.com/appscode/stash/pkg/restic"
	"github.com/appscode/stash/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDownloadCmd() *cobra.Command {
	var (
		kubeConfig     string
		repositoryName string
		namespace      string
		restoreOpt     = restic.RestoreOptions{
			SourceHost: restic.DefaultHost,
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

			// get source repository secret
			secret, err := c.kubeClient.CoreV1().Secrets(namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// cleanup whole scratch/secret dir at the end
			defer os.RemoveAll(cliScratchDir)
			defer os.RemoveAll(cliSecretDir)

			// write repository secrets in a temp dir
			if err := os.MkdirAll(cliSecretDir, 0755); err != nil {
				return err
			}
			for key, value := range secret.Data {
				if err := ioutil.WriteFile(filepath.Join(cliSecretDir, key), value, 0755); err != nil {
					return err
				}
			}

			// configure restic wrapper
			extraOpt := util.ExtraOptions{
				SecretDir:   cliSecretDir,
				EnableCache: false,
				ScratchDir:  cliScratchDir,
			}
			setupOpt, err := util.SetupOptionsForRepository(*repository, extraOpt)
			if err != nil {
				return fmt.Errorf("setup option for repository fail")
			}
			resticWrapper, err := restic.NewResticWrapper(setupOpt)
			if err != nil {
				return err
			}
			// if destination flag not specified, restore in current directory
			if restoreOpt.Destination == "" {
				restoreOpt.Destination, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			// run restore
			if _, err = resticWrapper.RunRestore(restoreOpt); err != nil {
				return err
			}
			log.Infof("Repository %s/%s restored in path %s", namespace, repositoryName, restoreOpt.Destination)
			return nil
		},
	}

	cmd.Flags().StringVar(&kubeConfig, "kubeconfig", kubeConfig, "Path of the Kube config file.")
	cmd.Flags().StringVar(&repositoryName, "repository", repositoryName, "Name of the Repository.")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "Namespace of the Repository.")

	cmd.Flags().StringVar(&restoreOpt.Destination, "destination", restoreOpt.Destination, "Destination path where snapshot will be restored.")
	cmd.Flags().StringVar(&restoreOpt.SourceHost, "host", restoreOpt.SourceHost, "Name of the source host machine")
	cmd.Flags().StringSliceVar(&restoreOpt.RestoreDirs, "directories", restoreOpt.RestoreDirs, "List of directories to be restored")
	// TODO: only allow a single snapshot ?
	cmd.Flags().StringSliceVar(&restoreOpt.Snapshots, "snapshots", restoreOpt.Snapshots, "List of snapshots to be restored")

	return cmd
}
