/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_util "kmodules.xyz/client-go/meta"
)

func (fi *Invocation) GetRestoreSession(repoName string, transformFuncs ...func(restore *v1beta1.RestoreSession)) *v1beta1.RestoreSession {
	restoreSession := &v1beta1.RestoreSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app),
			Namespace: fi.namespace,
		},
		Spec: v1beta1.RestoreSessionSpec{
			Repository: core.LocalObjectReference{
				Name: repoName,
			},
		},
	}
	// transformFuncs provides a array of functions that made test specific change on the RestoreSession
	// apply these test specific changes.
	for _, fn := range transformFuncs {
		fn(restoreSession)
	}
	return restoreSession
}

func (fi *Invocation) CreateRestoreSession(restoreSession *v1beta1.RestoreSession) error {
	_, err := fi.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Create(context.TODO(), restoreSession, metav1.CreateOptions{})
	return err
}

func (fi Invocation) DeleteRestoreSession(meta metav1.ObjectMeta) error {
	err := fi.StashClient.StashV1beta1().RestoreSessions(meta.Namespace).Delete(context.TODO(), meta.Name, metav1.DeleteOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyRestoreProcessCompleted(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			rs, err := f.StashClient.StashV1beta1().RestoreSessions(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if rs.Status.Phase == v1beta1.RestoreSessionSucceeded ||
				rs.Status.Phase == v1beta1.RestoreSessionFailed ||
				rs.Status.Phase == v1beta1.RestoreSessionUnknown {
				return true
			}
			return false
		},
	)
}

func (f *Framework) GetRestoreJob(restoreSessionName string) (*batchv1.Job, error) {
	return f.KubeClient.BatchV1().Jobs(f.namespace).Get(context.TODO(), getRestoreJobName(restoreSessionName), metav1.GetOptions{})
}

func getRestoreJobName(restoreSessionName string) string {
	return meta_util.ValidNameWithPrefix(apis.PrefixStashRestore, strings.ReplaceAll(restoreSessionName, ".", "-"))
}
