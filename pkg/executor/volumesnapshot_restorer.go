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
	"fmt"
	"strconv"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	"stash.appscode.dev/stash/pkg/rbac"

	"gomodules.xyz/flags"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/clientcmd"
	ofst_util "kmodules.xyz/offshoot-api/util"
)

type CSISnapshotRestorer struct {
	KubeClient       kubernetes.Interface
	Invoker          invoker.RestoreInvoker
	Index            int
	RBACOptions      *rbac.Options
	Image            docker.Docker
	ImagePullSecrets []core.LocalObjectReference
}

func (e *CSISnapshotRestorer) Ensure() (runtime.Object, kutil.VerbType, error) {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]
	runtimeSettings := targetInfo.RuntimeSettings

	jobMeta := metav1.ObjectMeta{
		Name:      e.getName(),
		Namespace: e.Invoker.GetObjectMeta().Namespace,
		Labels:    e.Invoker.GetLabels(),
	}

	err := e.RBACOptions.EnsureVolumeSnapshotRestorerJobRBAC()
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	jobTemplate := e.getJobTemplate()

	job := jobOptions{
		kubeClient:         e.KubeClient,
		meta:               jobMeta,
		owner:              e.Invoker.GetOwnerRef(),
		podSpec:            jobTemplate.Spec,
		podLabels:          e.Invoker.GetLabels(),
		serviceAccountName: e.RBACOptions.GetServiceAccountName(),
		backOffLimit:       0,
	}
	if runtimeSettings.Pod != nil && runtimeSettings.Pod.PodAnnotations != nil {
		job.podAnnotations = runtimeSettings.Pod.PodAnnotations
	}
	return job.ensure()
}

func (e *CSISnapshotRestorer) getName() string {
	return meta_util.ValidNameWithPrefixNSuffix(
		apis.PrefixStashVolumeSnapshot,
		strings.ReplaceAll(e.Invoker.GetObjectMeta().Name, ".", "-"),
		strconv.Itoa(e.Index),
	)
}

func (e *CSISnapshotRestorer) getJobTemplate() *core.PodTemplateSpec {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]

	container := core.Container{
		Name:  apis.StashContainer,
		Image: e.Image.ToContainerImage(),
		Args: append([]string{
			"restore-vs",
			"--invoker-kind=" + e.Invoker.GetTypeMeta().Kind,
			"--invoker-name=" + e.Invoker.GetObjectMeta().Name,
			"--target-name=" + targetInfo.Target.Ref.Name,
			"--target-namespace=" + targetInfo.Target.Ref.Namespace,
			"--target-kind=" + targetInfo.Target.Ref.Kind,
			"--metrics-enabled=true",
			"--pushgateway-url=" + metrics.GetPushgatewayURL(),
			fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
		}, flags.LoggerOptions.ToFlags()...),
		Env: []core.EnvVar{
			{
				Name: apis.KeyPodName,
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
	}

	// Pass container runtimeSettings from RestoreSession
	if targetInfo.RuntimeSettings.Container != nil {
		container = ofst_util.ApplyContainerRuntimeSettings(container, *targetInfo.RuntimeSettings.Container)
	}

	jobTemplate := &core.PodTemplateSpec{
		Spec: core.PodSpec{
			Containers:    []core.Container{container},
			RestartPolicy: core.RestartPolicyNever,
		},
	}

	// Pass pod runtimeSettings from RestoreSession
	if targetInfo.RuntimeSettings.Pod != nil {
		jobTemplate.Spec = ofst_util.ApplyPodRuntimeSettings(jobTemplate.Spec, *targetInfo.RuntimeSettings.Pod)
	}
	return jobTemplate
}
