/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"encoding/json"
	"fmt"

	cs "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1"
	"stash.appscode.dev/stash/pkg/registry/snapshot"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
)

func NewCmdSnapshots() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		repositoryName string
	)

	cmd := &cobra.Command{
		Use:               "snapshots [snapshotID ...]",
		Short:             "Get snapshots of restic repo",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				return err
			}

			stashClient := cs.NewForConfigOrDie(config)

			if repositoryName == "" {
				return fmt.Errorf("repository name not found")
			}
			repo, err := stashClient.Repositories(meta.Namespace()).Get(repositoryName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			r := snapshot.NewREST(config)
			snapshots, err := r.GetVersionedSnapshots(repo, args, true)
			if err != nil {
				return err
			}
			jsonSnaps, err := json.MarshalIndent(snapshots, "", "    ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsonSnaps))
			return nil
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&repositoryName, "repo-name", repositoryName, "Name of the Repository CRD.")

	return cmd
}
