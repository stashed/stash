package cli

import (
	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
)

func NewCopyRepositoryCmd() *cobra.Command {
	var (
		kubeConfig           string
		repositoryName       string
		sourceNamespace      string
		destinationNamespace string
	)

	var cmd = &cobra.Command{
		Use:               "copy-repository",
		Short:             `Copy Repository and Secret`,
		Long:              `Copy Repository and Secret from one namespace to another namespace`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "repository", "source-namespace", "destination-namespace")

			c, err := newStashCLIController(kubeConfig)
			if err != nil {
				return err
			}

			// get source repository
			repository, err := c.stashClient.StashV1alpha1().Repositories(sourceNamespace).Get(repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// get source repository secret
			secret, err := c.kubeClient.CoreV1().Secrets(sourceNamespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// for local backend create/patch PVC
			if repository.Spec.Backend.Local != nil && repository.Spec.Backend.Local.PersistentVolumeClaim != nil {
				// get PVC
				pvc, err := c.kubeClient.CoreV1().PersistentVolumeClaims(sourceNamespace).Get(
					repository.Spec.Backend.Local.PersistentVolumeClaim.ClaimName,
					metav1.GetOptions{},
				)
				if err != nil {
					return err
				}
				_, _, err = core_util.CreateOrPatchPVC(
					c.kubeClient,
					metav1.ObjectMeta{
						Name:      pvc.Name,
						Namespace: destinationNamespace,
					},
					func(obj *core.PersistentVolumeClaim) *core.PersistentVolumeClaim {
						obj.Spec = pvc.Spec
						return obj
					},
				)
				if err != nil {
					return err
				}
				log.Infof("PVC %s copied from namespace %s to %s", pvc.Name, sourceNamespace, destinationNamespace)
			}

			// create/patch destination repository secret
			// only copy data
			_, _, err = core_util.CreateOrPatchSecret(
				c.kubeClient,
				metav1.ObjectMeta{
					Name:      secret.Name,
					Namespace: destinationNamespace,
				},
				func(obj *core.Secret) *core.Secret {
					obj.Data = secret.Data
					return obj
				},
			)
			if err != nil {
				return err
			}
			log.Infof("Secret %s copied from namespace %s to %s", secret.Name, sourceNamespace, destinationNamespace)

			// create/patch destination repository
			// only copy spec
			_, _, err = util.CreateOrPatchRepository(
				c.stashClient.StashV1alpha1(),
				metav1.ObjectMeta{
					Name:      repository.Name,
					Namespace: destinationNamespace,
				},
				func(obj *v1alpha1.Repository) *v1alpha1.Repository {
					obj.Spec = repository.Spec
					return obj
				},
			)
			if err != nil {
				return err
			}
			log.Infof("Repository %s copied from namespace %s to %s", repositoryName, sourceNamespace, destinationNamespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&kubeConfig, "kubeconfig", kubeConfig, "Path of the Kube config file.")
	cmd.Flags().StringVar(&repositoryName, "repository", repositoryName, "Name of the Repository.")
	cmd.Flags().StringVar(&sourceNamespace, "source-namespace", sourceNamespace, "Source namespace.")
	cmd.Flags().StringVar(&destinationNamespace, "destination-namespace", destinationNamespace, "Destination namespace.")

	return cmd
}
