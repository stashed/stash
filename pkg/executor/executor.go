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

package executor

import (
	"context"

	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
	batch_util "kmodules.xyz/client-go/batch/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	metautil "kmodules.xyz/client-go/meta"
)

type Executor interface {
	Ensure() (runtime.Object, kutil.VerbType, error)
}

type Type string

const (
	TypeSidecar             Type = "Sidecar"
	TypeInitContainer       Type = "InitContainer"
	TypeBackupJob           Type = "BackupJob"
	TypeRestoreJob          Type = "RestoreJob"
	TypeCSISnapshooter      Type = "CSIVolumeSnapshooter"
	TypeCSISnapshotRestorer Type = "CSIVolumeRestorer"
)

type jobOptions struct {
	kubeClient         kubernetes.Interface
	meta               metav1.ObjectMeta
	owner              *metav1.OwnerReference
	podSpec            core.PodSpec
	podLabels          map[string]string
	podAnnotations     map[string]string
	imagePullSecrets   []core.LocalObjectReference
	serviceAccountName string
	backOffLimit       int32
}

func (opt jobOptions) ensure() (runtime.Object, kutil.VerbType, error) {
	return batch_util.CreateOrPatchJob(
		context.TODO(),
		opt.kubeClient,
		opt.meta,
		func(in *batch.Job) *batch.Job {
			core_util.EnsureOwnerReference(&in.ObjectMeta, opt.owner)

			in.Spec.Template.Spec = opt.upsertPodSpec(in.Spec.Template.Spec)
			in.Spec.Template.Annotations = opt.podAnnotations
			in.Spec.Template.Labels = metautil.OverwriteKeys(in.Spec.Template.Labels, opt.podLabels)
			in.Spec.Template.Spec.ImagePullSecrets = core_util.MergeLocalObjectReferences(in.Spec.Template.Spec.ImagePullSecrets, opt.imagePullSecrets)
			in.Spec.Template.Spec.ServiceAccountName = opt.serviceAccountName
			in.Spec.BackoffLimit = &opt.backOffLimit
			return in
		},
		metav1.PatchOptions{},
	)
}

func (opt jobOptions) upsertPodSpec(cur core.PodSpec) core.PodSpec {
	cur.Volumes = core_util.UpsertVolume(cur.Volumes, opt.podSpec.Volumes...)
	cur.Containers = core_util.UpsertContainers(cur.Containers, opt.podSpec.Containers)
	cur.InitContainers = core_util.UpsertContainers(cur.InitContainers, opt.podSpec.InitContainers)
	cur.ServiceAccountName = opt.serviceAccountName
	cur.SecurityContext = opt.podSpec.SecurityContext
	cur.ImagePullSecrets = opt.podSpec.ImagePullSecrets
	cur.RestartPolicy = opt.podSpec.RestartPolicy
	if opt.podSpec.NodeName != "" {
		cur.NodeName = opt.podSpec.NodeName
	}
	if opt.podSpec.NodeSelector != nil {
		cur.NodeSelector = opt.podSpec.NodeSelector
	}
	return cur
}
