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
	"fmt"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/stash/pkg/rbac"

	"gomodules.xyz/pointer"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	batchutil "kmodules.xyz/client-go/batch"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	ofst_util "kmodules.xyz/offshoot-api/util"
)

type PeriodicScheduler struct {
	KubeClient       kubernetes.Interface
	Image            docker.Docker
	Invoker          invoker.BackupInvoker
	RBACOptions      *rbac.Options
	ImagePullSecrets []core.LocalObjectReference
}

func (s *PeriodicScheduler) Ensure() error {
	invMeta := s.Invoker.GetObjectMeta()
	runtimeSettings := s.Invoker.GetRuntimeSettings()
	ownerRef := s.Invoker.GetOwnerRef()

	cronMeta := metav1.ObjectMeta{
		Name:      s.generateName(),
		Namespace: invMeta.Namespace,
		Labels:    s.Invoker.GetLabels(),
	}

	err := s.RBACOptions.EnsureCronJobRBAC(cronMeta.Name)
	if err != nil {
		return err
	}

	_, _, err = batchutil.CreateOrPatchCronJob(
		context.TODO(),
		s.KubeClient,
		cronMeta,
		func(in *batch.CronJob) *batch.CronJob {
			// set backup invoker object as cron-job owner
			core_util.EnsureOwnerReference(&in.ObjectMeta, ownerRef)

			in.Spec.Schedule = s.Invoker.GetSchedule()
			in.Spec.Suspend = pointer.BoolP(s.Invoker.IsPaused()) // this ensure that the CronJob is suspended when the backup invoker is paused.
			in.Spec.JobTemplate.Labels = meta_util.OverwriteKeys(in.Spec.JobTemplate.Labels, s.Invoker.GetLabels())
			// ensure that job gets deleted on completion
			in.Spec.JobTemplate.Labels[apis.KeyDeleteJobOnCompletion] = apis.AllowDeletingJobOnCompletion
			// pass offshoot labels to the CronJob's pod
			in.Spec.JobTemplate.Spec.Template.Labels = meta_util.OverwriteKeys(in.Spec.JobTemplate.Spec.Template.Labels, s.Invoker.GetLabels())

			container := core.Container{
				Name:            apis.StashCronJobContainer,
				ImagePullPolicy: core.PullIfNotPresent,
				Image:           s.Image.ToContainerImage(),
				Args: []string{
					"create-backupsession",
					fmt.Sprintf("--invoker-name=%s", ownerRef.Name),
					fmt.Sprintf("--invoker-kind=%s", ownerRef.Kind),
				},
			}
			// only apply the container level runtime settings that make sense for the CronJob
			if runtimeSettings.Container != nil {
				container.Resources = runtimeSettings.Container.Resources
				container.Env = runtimeSettings.Container.Env
				container.EnvFrom = runtimeSettings.Container.EnvFrom
				container.SecurityContext = runtimeSettings.Container.SecurityContext
			}

			in.Spec.JobTemplate.Spec.Template.Spec.Containers = core_util.UpsertContainer(
				in.Spec.JobTemplate.Spec.Template.Spec.Containers, container)
			in.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
			in.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = s.RBACOptions.GetServiceAccountName()
			in.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets = s.ImagePullSecrets

			// apply the pod level runtime settings to the CronJob
			if runtimeSettings.Pod != nil {
				in.Spec.JobTemplate.Spec.Template.Spec = ofst_util.ApplyPodRuntimeSettings(in.Spec.JobTemplate.Spec.Template.Spec, *runtimeSettings.Pod)
			}

			return in
		},
		metav1.PatchOptions{},
	)

	return err
}

func (s *PeriodicScheduler) Cleanup() error {
	invMeta := s.Invoker.GetObjectMeta()
	cur, err := batchutil.GetCronJob(context.TODO(), s.KubeClient, types.NamespacedName{Namespace: invMeta.Namespace, Name: s.generateName()})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil
		}
		return err
	}
	_, _, err = batchutil.PatchCronJob(
		context.TODO(),
		s.KubeClient,
		cur,
		func(in *batch.CronJob) *batch.CronJob {
			core_util.EnsureOwnerReference(&in.ObjectMeta, s.Invoker.GetOwnerRef())
			return in
		},
		metav1.PatchOptions{},
	)
	if err != nil {
		return err
	}
	return s.cleanupRBAC()
}

func (s *PeriodicScheduler) cleanupRBAC() error {
	return s.RBACOptions.EnsureRBACResourcesDeleted()
}

func (s *PeriodicScheduler) generateName() string {
	invMeta := s.Invoker.GetObjectMeta()
	if s.getTargetNamespace() != invMeta.Namespace {
		return meta_util.ValidCronJobNameWithPrefixNSuffix(apis.PrefixStashTrigger, invMeta.Namespace, strings.ReplaceAll(invMeta.Name, ".", "-"))
	}
	return meta_util.ValidCronJobNameWithSuffix(apis.PrefixStashTrigger, strings.ReplaceAll(invMeta.Name, ".", "-"))
}

func (s *PeriodicScheduler) getTargetNamespace() string {
	for _, t := range s.Invoker.GetTargetInfo() {
		if t.Target != nil && t.Target.Ref.Namespace != "" {
			return t.Target.Ref.Namespace
		}
	}
	return s.Invoker.GetObjectMeta().Namespace
}
