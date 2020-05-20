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

package framework

import (
	"time"

	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
)

func (f *Framework) EventuallyCondition(meta metav1.ObjectMeta, kind string, condType string) GomegaAsyncAssertion {
	return Eventually(
		func() kmapi.ConditionStatus {
			var conditions []kmapi.Condition
			switch kind {
			case v1beta1.ResourceKindBackupConfiguration:
				bc, err := f.StashClient.StashV1beta1().BackupConfigurations(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
				if err != nil {
					return kmapi.ConditionUnknown
				}
				conditions = bc.Status.Conditions
			case v1beta1.ResourceKindBackupBatch:
				bb, err := f.StashClient.StashV1beta1().BackupBatches(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
				if err != nil {
					return kmapi.ConditionUnknown
				}
				conditions = bb.Status.Conditions
			case v1beta1.ResourceKindRestoreSession:
				rs, err := f.StashClient.StashV1beta1().RestoreSessions(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
				if err != nil {
					return kmapi.ConditionUnknown
				}
				conditions = rs.Status.Conditions
			}
			_, cond := kmapi.GetCondition(conditions, condType)
			if cond == nil {
				return kmapi.ConditionUnknown
			}
			return cond.Status
		},
		5*time.Minute,
		2*time.Second,
	)
}

func (f *Framework) EventuallyTargetCondition(meta metav1.ObjectMeta, target v1beta1.TargetRef, condType v1beta1.BackupInvokerCondition) GomegaAsyncAssertion {
	return Eventually(
		func() kmapi.ConditionStatus {
			bb, err := f.StashClient.StashV1beta1().BackupBatches(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			if err != nil {
				return kmapi.ConditionUnknown
			}
			for _, mc := range bb.Status.MemberConditions {
				if mc.Target.APIVersion == target.APIVersion &&
					mc.Target.Kind == target.Kind &&
					mc.Target.Name == target.Name {
					_, cond := kmapi.GetCondition(mc.Conditions, string(condType))
					if cond != nil {
						return cond.Status
					}
					return kmapi.ConditionUnknown

				}
			}
			return kmapi.ConditionUnknown
		},
		5*time.Minute,
		2*time.Second,
	)
}
