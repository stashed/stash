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
	"fmt"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	. "github.com/onsi/gomega"
	"gomodules.xyz/x/arrays"
	"gomodules.xyz/x/crypto/rand"
	batchv1 "k8s.io/api/batch/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	kmapi "kmodules.xyz/client-go/api/v1"
	meta_util "kmodules.xyz/client-go/meta"
)

func (fi *Invocation) GetRestoreSession(repoName string, transformFuncs ...func(restore *v1beta1.RestoreSession)) *v1beta1.RestoreSession {
	restoreSession := &v1beta1.RestoreSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app),
			Namespace: fi.namespace,
		},
		Spec: v1beta1.RestoreSessionSpec{
			Repository: kmapi.ObjectReference{
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

func (f *Framework) EventuallyRestoreProcessCompleted(meta metav1.ObjectMeta, invokerKind string) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			var restorePhase v1beta1.RestorePhase
			if invokerKind == v1beta1.ResourceKindRestoreSession {
				rs, err := f.StashClient.StashV1beta1().RestoreSessions(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
				if err != nil {
					return false
				}
				restorePhase = rs.Status.Phase
			} else {
				rb, err := f.StashClient.StashV1beta1().RestoreBatches(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
				if err != nil {
					return false
				}
				restorePhase = rb.Status.Phase
			}
			if restorePhase == v1beta1.RestoreSucceeded ||
				restorePhase == v1beta1.RestoreFailed ||
				restorePhase == v1beta1.RestorePhaseUnknown {
				return true
			}
			return false
		}, WaitTimeOut, PullInterval)
}

func (f *Framework) GetRestoreJobs() ([]batchv1.Job, error) {
	selector := labels.SelectorFromSet(map[string]string{
		meta_util.ComponentLabelKey: v1beta1.StashRestoreComponent,
		meta_util.ManagedByLabelKey: apis.StashKey,
	})
	var restoreJobs []batchv1.Job
	jobs, err := f.KubeClient.BatchV1().Jobs(f.namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	for i := range jobs.Items {
		if strings.HasPrefix(jobs.Items[i].ObjectMeta.Name, apis.PrefixStashRestore) {
			restoreJobs = append(restoreJobs, jobs.Items[i])
		}
	}
	return restoreJobs, nil
}

func JobsTargetMatch(job batchv1.Job, targetRef v1beta1.TargetRef) bool {
	containers := append(job.Spec.Template.Spec.InitContainers, job.Spec.Template.Spec.Containers...)
	for _, c := range containers {
		targetKindMatched, _ := arrays.Contains(c.Args, fmt.Sprintf("--target-kind=%s", targetRef.Kind))
		targetNameMatched, _ := arrays.Contains(c.Args, fmt.Sprintf("--target-name=%s", targetRef.Name))
		if targetKindMatched && targetNameMatched {
			return true
		}
	}
	return false
}

func RulesMigrated(restoreSession *v1beta1.RestoreSession, rules []v1beta1.Rule) bool {
	if len(restoreSession.Spec.Rules) != 0 || len(restoreSession.Spec.Target.Rules) == 0 {
		return false
	}
	for _, rule := range restoreSession.Spec.Target.Rules {
		if !ruleExist(rule, rules) {
			return false
		}
	}

	return true
}

func ruleExist(rule v1beta1.Rule, rules []v1beta1.Rule) bool {
	for i := range rules {
		if rules[i].SourceHost == rule.SourceHost &&
			sets.NewString(rules[i].Paths...).Equal(sets.NewString(rule.Paths...)) &&
			sets.NewString(rules[i].Include...).Equal(sets.NewString(rule.Include...)) &&
			sets.NewString(rules[i].Exclude...).Equal(sets.NewString(rule.Exclude...)) &&
			sets.NewString(rules[i].Snapshots...).Equal(sets.NewString(rule.Snapshots...)) &&
			sets.NewString(rules[i].TargetHosts...).Equal(sets.NewString(rule.TargetHosts...)) {
			return true
		}
	}
	return false
}

func (f *Framework) EventuallyRestoreInvokerPhase(invoker invoker.RestoreInvoker) GomegaAsyncAssertion {
	return Eventually(
		func() v1beta1.RestorePhase {
			switch invoker.GetTypeMeta().Kind {
			case v1beta1.ResourceKindRestoreSession:
				rs, err := f.StashClient.StashV1beta1().RestoreSessions(invoker.GetObjectMeta().Namespace).Get(context.TODO(), invoker.GetObjectMeta().Name, metav1.GetOptions{})
				if err != nil {
					return ""
				}
				return rs.Status.Phase
			case v1beta1.ResourceKindRestoreBatch:
				rb, err := f.StashClient.StashV1beta1().RestoreBatches(invoker.GetObjectMeta().Namespace).Get(context.TODO(), invoker.GetObjectMeta().Name, metav1.GetOptions{})
				if err != nil {
					return ""
				}
				return rb.Status.Phase
			default:
				return ""
			}
		}, WaitTimeOut, PullInterval)
}

func (fi *Invocation) TargetRestoreExecuted(inv invoker.RestoreInvoker) bool {
	for _, t := range inv.GetStatus().TargetStatus {
		if len(t.Stats) > 0 {
			return true
		}
	}
	return false
}
