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

package util

import (
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/pkg/docker"

	"github.com/appscode/go/types"
	core "k8s.io/api/core/v1"
	"kmodules.xyz/client-go/tools/analytics"
	"kmodules.xyz/client-go/tools/cli"
	"kmodules.xyz/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/tools/pushgateway"
	store "kmodules.xyz/objectstore-api/api/v1"
	ofst_util "kmodules.xyz/offshoot-api/util"
)

func NewSidecarContainer(r *api.Restic, workload api.LocalTypedReference, image docker.Docker) core.Container {
	if r.Annotations != nil {
		if v, ok := r.Annotations[apis.VersionTag]; ok {
			image.Tag = v
		}
	}
	sidecar := core.Container{
		Name:  apis.StashContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"backup",
			"--restic-name=" + r.Name,
			"--workload-kind=" + workload.Kind,
			"--workload-name=" + workload.Name,
			"--docker-registry=" + image.Registry,
			"--image-tag=" + image.Tag,
			"--run-via-cron=true",
			"--pushgateway-url=" + pushgateway.URL(),
			fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
			fmt.Sprintf("--enable-analytics=%v", cli.EnableAnalytics),
		}, cli.LoggerOptions.ToFlags()...),
		Env: []core.EnvVar{
			{
				Name: "NODE_NAME",
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			{
				Name: "POD_NAME",
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name:  analytics.Key,
				Value: cli.AnalyticsClientID,
			},
		},
		Resources: r.Spec.Resources,
		SecurityContext: &core.SecurityContext{
			RunAsUser:  types.Int64P(0),
			RunAsGroup: types.Int64P(0),
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      apis.ScratchDirVolumeName,
				MountPath: "/tmp",
			},
			{
				Name:      apis.PodinfoVolumeName,
				MountPath: "/etc/stash",
			},
		},
	}
	for _, srcVol := range r.Spec.VolumeMounts {
		sidecar.VolumeMounts = append(sidecar.VolumeMounts, core.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
		})
	}
	if r.Spec.Backend.Local != nil {
		_, mnt := r.Spec.Backend.Local.ToVolumeAndMount(apis.LocalVolumeName)
		sidecar.VolumeMounts = append(sidecar.VolumeMounts, mnt)
	}
	return sidecar
}

func NewBackupSidecarContainer(invoker apis.Invoker, targetInfo apis.TargetInfo, backend *store.Backend, image docker.Docker) core.Container {
	sidecar := core.Container{
		Name:  apis.StashContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"run-backup",
			"--invoker-name=" + invoker.ObjectMeta.Name,
			"--invoker-type=" + invoker.ObjectRef.Kind,
			"--target-name=" + targetInfo.Target.Ref.Name,
			"--target-kind=" + targetInfo.Target.Ref.Kind,
			"--secret-dir=" + apis.StashSecretMountDir,
			fmt.Sprintf("--enable-cache=%v", !targetInfo.TempDir.DisableCaching),
			fmt.Sprintf("--max-connections=%v", backend.MaxConnections()),
			"--metrics-enabled=true",
			"--pushgateway-url=" + pushgateway.URL(),
			fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
			fmt.Sprintf("--enable-analytics=%v", cli.EnableAnalytics),
		}, cli.LoggerOptions.ToFlags()...),
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
		VolumeMounts: []core.VolumeMount{
			{
				Name:      apis.PodinfoVolumeName,
				MountPath: "/etc/stash",
			},
			{
				Name:      apis.StashSecretVolume,
				MountPath: apis.StashSecretMountDir,
			},
		},
	}

	// mount tmp volume
	sidecar.VolumeMounts = UpsertTmpVolumeMount(sidecar.VolumeMounts)

	// mount the volumes specified in invoker this sidecar
	for _, srcVol := range targetInfo.Target.VolumeMounts {
		sidecar.VolumeMounts = append(sidecar.VolumeMounts, core.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
		})
	}
	// if Repository uses local volume as backend, we have to mount it inside the sidecar
	if backend.Local != nil {
		_, mnt := backend.Local.ToVolumeAndMount(apis.LocalVolumeName)
		sidecar.VolumeMounts = append(sidecar.VolumeMounts, mnt)
	}
	// pass container runtime settings from invoker to sidecar
	if targetInfo.RuntimeSettings.Container != nil {
		sidecar = ofst_util.ApplyContainerRuntimeSettings(sidecar, *targetInfo.RuntimeSettings.Container)
	}
	return sidecar
}
