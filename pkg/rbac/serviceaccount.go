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

package rbac

import (
	"context"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

func (opt *Options) ensureServiceAccount() error {
	saMeta := metav1.ObjectMeta{
		Name:        opt.serviceAccount.Name,
		Namespace:   opt.serviceAccount.Namespace,
		Labels:      opt.offshootLabels,
		Annotations: opt.serviceAccount.Annotations,
	}
	_, _, err := core_util.CreateOrPatchServiceAccount(
		context.TODO(),
		opt.kubeClient,
		saMeta,
		func(in *core.ServiceAccount) *core.ServiceAccount {
			core_util.EnsureOwnerReference(&in.ObjectMeta, opt.owner)
			return in
		},
		metav1.PatchOptions{},
	)

	return err
}
