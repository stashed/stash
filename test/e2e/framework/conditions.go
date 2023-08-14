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

	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
	condutil "kmodules.xyz/client-go/conditions"
)

func (f *Framework) EventuallyCondition(meta metav1.ObjectMeta, kind string, condType string) GomegaAsyncAssertion {
	return Eventually(
		func() metav1.ConditionStatus {
			var conditions []kmapi.Condition
			switch kind {
			case v1beta1.ResourceKindBackupConfiguration:
				bc, err := f.StashClient.StashV1beta1().BackupConfigurations(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
				if err != nil {
					return metav1.ConditionUnknown
				}
				conditions = bc.Status.Conditions
			case v1beta1.ResourceKindRestoreSession:
				rs, err := f.StashClient.StashV1beta1().RestoreSessions(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
				if err != nil {
					return metav1.ConditionUnknown
				}
				conditions = rs.Status.Conditions
			}
			_, cond := condutil.GetCondition(conditions, condType)
			if cond == nil {
				return metav1.ConditionUnknown
			}
			return cond.Status
		},
		WaitTimeOut,
		PullInterval,
	)
}
