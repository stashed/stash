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

package framework

import (
	"context"

	snap_v1alpha1 "stash.appscode.dev/apimachinery/apis/repositories/v1alpha1"

	"github.com/appscode/go/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (fi *Invocation) ListSnapshots(selector string) (*snap_v1alpha1.SnapshotList, error) {
	return fi.StashClient.RepositoriesV1alpha1().Snapshots(fi.Namespace()).List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector:  selector,
			TimeoutSeconds: types.Int64P(600),
		})
}

func (fi *Invocation) GetSnapshot(name string) (*snap_v1alpha1.Snapshot, error) {
	return fi.StashClient.RepositoriesV1alpha1().Snapshots(fi.Namespace()).Get(context.TODO(), name, metav1.GetOptions{})
}

func (fi *Invocation) DeleteSnapshot(name string) error {
	policy := metav1.DeletePropagationForeground
	return fi.StashClient.RepositoriesV1alpha1().Snapshots(fi.Namespace()).Delete(
		context.TODO(),
		name,
		metav1.DeleteOptions{
			PropagationPolicy: &policy,
		},
	)
}
