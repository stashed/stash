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
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"

	"github.com/appscode/go/crypto/rand"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (fi *Invocation) BackupBatch(repoName string) *v1beta1.BackupBatch {

	return &v1beta1.BackupBatch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app),
			Namespace: fi.namespace,
		},
		Spec: v1beta1.BackupBatchSpec{
			// some workloads such as StatefulSet or DaemonSet may take long to complete backup. so, giving a fixed short interval is not always feasible.
			// hence, set the schedule to a large interval so that no backup schedule appear before completing running backup
			// we will use manual triggering for taking backup. this will help us to avoid waiting for fixed interval before each backup
			// and the problem mentioned above
			Schedule: "59 * * * *",
			Repository: core.LocalObjectReference{
				Name: repoName,
			},
			RetentionPolicy: v1alpha1.RetentionPolicy{
				Name:     "keep-last-10",
				KeepLast: 10,
				Prune:    true,
			},
		},
	}
}
