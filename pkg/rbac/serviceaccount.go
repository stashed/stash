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
	"strings"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
)

func (opt *RBACOptions) ensureServiceAccount() error {
	// ServiceAccount hasn't been specified. so create new one.
	opt.ServiceAccount.Name = meta.NameWithSuffix(strings.ToLower(opt.Invoker.Kind), opt.Invoker.Name)

	saMeta := metav1.ObjectMeta{
		Name:      opt.ServiceAccount.Name,
		Namespace: opt.ServiceAccount.Namespace,
		Labels:    opt.OffshootLabels,
	}
	_, _, err := core_util.CreateOrPatchServiceAccount(
		context.TODO(),
		opt.KubeClient,
		saMeta,
		func(in *core.ServiceAccount) *core.ServiceAccount {
			core_util.EnsureOwnerReference(&in.ObjectMeta, opt.Owner)
			return in
		},
		metav1.PatchOptions{},
	)

	return err
}
