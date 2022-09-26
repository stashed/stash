/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"context"
	"fmt"

	cs "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1"
	"stash.appscode.dev/stash/pkg/registry/snapshot"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kmapi "kmodules.xyz/client-go/api/v1"
)

func NewCmdForget() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		repo           kmapi.ObjectReference
	)

	cmd := &cobra.Command{
		Use:               "forget [snapshotID ...]",
		Short:             "Delete snapshots from a restic repository",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				return err
			}

			stashClient := cs.NewForConfigOrDie(config)
			kubeClient := kubernetes.NewForConfigOrDie(config)

			if repo.Name == "" {
				return fmt.Errorf("repository name not found")
			}
			repo, err := stashClient.Repositories(repo.Namespace).Get(context.TODO(), repo.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			secret, err := kubeClient.CoreV1().Secrets(repo.Namespace).Get(context.TODO(), repo.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			opt := snapshot.Options{
				Repository:  repo,
				Secret:      secret,
				SnapshotIDs: args,
				InCluster:   true,
			}

			r := snapshot.NewREST(config)
			return r.ForgetSnapshotsFromBackend(opt)
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&repo.Name, "repo-name", repo.Name, "Name of the Repository CRD.")
	cmd.Flags().StringVar(&repo.Namespace, "repo-namespace", repo.Namespace, "Namespace of the Repository CRD.")

	return cmd
}
