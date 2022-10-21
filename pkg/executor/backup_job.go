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
	"strconv"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/resolver"
	"stash.appscode.dev/stash/pkg/util"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
	metautil "kmodules.xyz/client-go/meta"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

type BackupJob struct {
	KubeClient        kubernetes.Interface
	StashClient       cs.Interface
	CatalogClient     appcatalog_cs.Interface
	RBACOptions       *rbac.Options
	ImagePullSecrets  []core.LocalObjectReference
	Invoker           invoker.BackupInvoker
	Session           *invoker.BackupSessionHandler
	Index             int
	Repository        *v1alpha1.Repository
	LicenseApiService string
	Image             docker.Docker
}

func (e *BackupJob) Ensure() (runtime.Object, kutil.VerbType, error) {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]
	runtimeSettings := targetInfo.RuntimeSettings

	jobMeta := metav1.ObjectMeta{
		Name:      e.getBackupJobName(),
		Namespace: e.Session.GetObjectMeta().Namespace,
		Labels:    e.Invoker.GetLabels(),
	}

	err := e.RBACOptions.EnsureBackupJobRBAC()
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	ownerBackupSession := metav1.NewControllerRef(e.Session.GetBackupSession(), api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupSession))
	podSpec, err := e.resolveTask(jobMeta, ownerBackupSession)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	job := &jobOptions{
		kubeClient:         e.KubeClient,
		meta:               jobMeta,
		owner:              ownerBackupSession,
		podSpec:            podSpec,
		podLabels:          e.Invoker.GetLabels(),
		serviceAccountName: e.RBACOptions.GetServiceAccountName(),
		imagePullSecrets:   e.ImagePullSecrets,
		runtimeSettings:    runtimeSettings,
		backOffLimit:       0,
	}
	if runtimeSettings.Pod != nil && runtimeSettings.Pod.PodAnnotations != nil {
		job.podAnnotations = runtimeSettings.Pod.PodAnnotations
	}
	return job.ensure()
}

func (e *BackupJob) resolveTask(jobMeta metav1.ObjectMeta, owner *metav1.OwnerReference) (core.PodSpec, error) {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]

	r := resolver.TaskOptions{
		StashClient:       e.StashClient,
		CatalogClient:     e.CatalogClient,
		Repository:        e.Repository,
		Image:             e.Image,
		LicenseApiService: e.LicenseApiService,
		Backup: &resolver.BackupOptions{
			Invoker:    e.Invoker,
			Session:    e.Session,
			TargetInfo: targetInfo,
		},
	}
	podSpec, err := r.Resolve()
	if err != nil {
		return core.PodSpec{}, err
	}

	// upsert InterimVolume to hold the backup/restored data temporarily
	return util.UpsertInterimVolume(
		e.KubeClient,
		podSpec,
		targetInfo.InterimVolumeTemplate.ToCorePVC(),
		jobMeta.Namespace,
		owner,
	)
}

func (e *BackupJob) getBackupJobName() string {
	return metautil.ValidNameWithPrefixNSuffix(
		apis.PrefixStashBackup,
		strings.ReplaceAll(e.Session.GetObjectMeta().Name, ".", "-"), strconv.Itoa(e.Index))
}
