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

package rbac

import (
	"context"
	"fmt"

	"stash.appscode.dev/apimachinery/apis"

	"gomodules.xyz/pointer"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	rbac_util "kmodules.xyz/client-go/rbac/v1"
)

func (opt *RBACOptions) ensureLicenseReaderClusterRoleBinding() error {
	meta := metav1.ObjectMeta{
		Name:   meta_util.NameWithSuffix(apis.LicenseReader, fmt.Sprintf("%s-%s", opt.ServiceAccount.Namespace, opt.ServiceAccount.Name)),
		Labels: opt.OffshootLabels,
	}
	owner := opt.Owner
	owner.Controller = pointer.BoolP(false)
	_, _, err := rbac_util.CreateOrPatchClusterRoleBinding(context.TODO(), opt.KubeClient, meta, func(in *rbac.ClusterRoleBinding) *rbac.ClusterRoleBinding {
		core_util.EnsureOwnerReference(&in.ObjectMeta, owner)

		in.RoleRef = rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     apis.KindClusterRole,
			Name:     apis.LicenseReader,
		}
		in.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      opt.ServiceAccount.Name,
				Namespace: opt.ServiceAccount.Namespace,
			},
		}
		return in
	}, metav1.PatchOptions{})
	return err
}
