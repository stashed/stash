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

package util

import (
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"

	"gomodules.xyz/flags"
	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	"kmodules.xyz/client-go/tools/clientcmd"
	v1 "kmodules.xyz/offshoot-api/api/v1"
	ofst_util "kmodules.xyz/offshoot-api/util"
)

// NewPVCRestorerJob return a job definition to restore pvc.
func NewPVCRestorerJob(inv invoker.RestoreInvoker, index int, repository *api_v1alpha1.Repository, image docker.Docker) (*core.PodTemplateSpec, error) {
	targetInfo := inv.GetTargetInfo()[index]
	container := core.Container{
		Name:  apis.StashContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"restore",
			"--invoker-kind=" + inv.GetTypeMeta().Kind,
			"--invoker-name=" + inv.GetObjectMeta().Name,
			"--target-kind=" + targetInfo.Target.Ref.Kind,
			"--target-name=" + targetInfo.Target.Ref.Name,
			"--target-namespace=" + targetInfo.Target.Ref.Namespace,
			"--restore-model=job",
			fmt.Sprintf("--enable-cache=%v", !targetInfo.TempDir.DisableCaching),
			fmt.Sprintf("--max-connections=%v", repository.Spec.Backend.MaxConnections()),
			"--metrics-enabled=true",
			"--pushgateway-url=" + metrics.GetPushgatewayURL(),
			fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
		}, flags.LoggerOptions.ToFlags()...),
		Env: []core.EnvVar{
			{
				Name: apis.KeyNodeName,
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
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

	// mount tmp volume
	container.VolumeMounts = UpsertTmpVolumeMount(container.VolumeMounts)

	// mount the volumes specified in RestoreSession into the job
	for _, srcVol := range targetInfo.Target.VolumeMounts {
		container.VolumeMounts = append(container.VolumeMounts, core.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
		})
	}

	// Pass container RuntimeSettings from RestoreSession
	if targetInfo.RuntimeSettings.Container != nil {
		container = ofst_util.ApplyContainerRuntimeSettings(container, *targetInfo.RuntimeSettings.Container)
	}

	// In order to preserve file ownership, restore process need to be run as root user.
	// Stash image uses non-root user 65535. We have to use securityContext to run stash as root user.
	// If a user specify securityContext either in pod level or container level in RuntimeSetting,
	// don't overwrite that. In this case, user must take the responsibility of possible file ownership modification.
	securityContext := &core.SecurityContext{
		RunAsUser:  pointer.Int64P(0),
		RunAsGroup: pointer.Int64P(0),
	}
	if targetInfo.RuntimeSettings.Container != nil {
		container.SecurityContext = UpsertSecurityContext(securityContext, targetInfo.RuntimeSettings.Container.SecurityContext)
	} else {
		container.SecurityContext = securityContext
	}

	jobTemplate := &core.PodTemplateSpec{
		Spec: core.PodSpec{
			Containers:    []core.Container{container},
			RestartPolicy: core.RestartPolicyNever,
		},
	}

	// Pass pod RuntimeSettings from RestoreSession
	if targetInfo.RuntimeSettings.Pod != nil {
		jobTemplate.Spec = ofst_util.ApplyPodRuntimeSettings(jobTemplate.Spec, *targetInfo.RuntimeSettings.Pod)
	}

	// add an emptyDir volume for holding temporary files
	jobTemplate.Spec.Volumes = UpsertTmpVolume(jobTemplate.Spec.Volumes, targetInfo.TempDir)

	return jobTemplate, nil
}

func NewVolumeSnapshotterJob(session *invoker.BackupSessionHandler, backupTarget *api_v1beta1.BackupTarget, runtimeSettings v1.RuntimeSettings, image docker.Docker) (*core.PodTemplateSpec, error) {
	container := core.Container{
		Name:  apis.StashContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"create-vs",
			fmt.Sprintf("--backupsession=%s", session.GetObjectMeta().Name),
			"--target-name=" + backupTarget.Ref.Name,
			"--target-namespace=" + backupTarget.Ref.Namespace,
			"--target-kind=" + backupTarget.Ref.Kind,
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

	// Pass container RuntimeSettings from RestoreSession
	if runtimeSettings.Container != nil {
		container = ofst_util.ApplyContainerRuntimeSettings(container, *runtimeSettings.Container)
	}

	jobTemplate := &core.PodTemplateSpec{
		Spec: core.PodSpec{
			Containers:    []core.Container{container},
			RestartPolicy: core.RestartPolicyNever,
		},
	}

	// Pass pod RuntimeSettings from RestoreSession
	if runtimeSettings.Pod != nil {
		jobTemplate.Spec = ofst_util.ApplyPodRuntimeSettings(jobTemplate.Spec, *runtimeSettings.Pod)
	}
	return jobTemplate, nil
}

func NewVolumeRestorerJob(inv invoker.RestoreInvoker, index int, image docker.Docker) (*core.PodTemplateSpec, error) {
	targetInfo := inv.GetTargetInfo()[index]

	container := core.Container{
		Name:  apis.StashContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"restore-vs",
			"--invoker-kind=" + inv.GetTypeMeta().Kind,
			"--invoker-name=" + inv.GetObjectMeta().Name,
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

	// Pass container RuntimeSettings from RestoreSession
	if targetInfo.RuntimeSettings.Container != nil {
		container = ofst_util.ApplyContainerRuntimeSettings(container, *targetInfo.RuntimeSettings.Container)
	}

	jobTemplate := &core.PodTemplateSpec{
		Spec: core.PodSpec{
			Containers:    []core.Container{container},
			RestartPolicy: core.RestartPolicyNever,
		},
	}

	// Pass pod RuntimeSettings from RestoreSession
	if targetInfo.RuntimeSettings.Pod != nil {
		jobTemplate.Spec = ofst_util.ApplyPodRuntimeSettings(jobTemplate.Spec, *targetInfo.RuntimeSettings.Pod)
	}
	return jobTemplate, nil
}
