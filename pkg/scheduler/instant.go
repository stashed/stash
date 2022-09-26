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

package scheduler

import (
	"context"

	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	stash_cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

type InstantScheduler struct {
	StashClient stash_cs.Interface
	Invoker     invoker.BackupInvoker
	RetryLeft   int32
}

func (s *InstantScheduler) Ensure() error {
	session := s.Invoker.NewSession()

	_, _, err := v1beta1_util.CreateOrPatchBackupSession(
		context.TODO(),
		s.StashClient.StashV1beta1(),
		session.ObjectMeta,
		func(in *api_v1beta1.BackupSession) *api_v1beta1.BackupSession {
			core_util.EnsureOwnerReference(&in.ObjectMeta, s.Invoker.GetOwnerRef())
			in.Labels = session.Labels
			in.Spec = session.Spec
			in.Spec.RetryLeft = s.RetryLeft
			return in
		},
		metav1.PatchOptions{},
	)
	return err
}
