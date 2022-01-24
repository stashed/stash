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
	v1alpha1_api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	"gomodules.xyz/flags"
	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	"kmodules.xyz/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/tools/pushgateway"
	ofst_util "kmodules.xyz/offshoot-api/util"
)

func NewRestoreInitContainer(inv invoker.RestoreInvoker, targetInfo invoker.RestoreTargetInfo, repository *v1alpha1_api.Repository, image docker.Docker) core.Container {
	initContainer := core.Container{
		Name:  apis.StashInitContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"restore",
			"--invoker-kind=" + inv.GetTypeMeta().Kind,
			"--invoker-name=" + inv.GetObjectMeta().Name,
			"--target-name=" + targetInfo.Target.Ref.Name,
			"--target-kind=" + targetInfo.Target.Ref.Kind,
			fmt.Sprintf("--enable-cache=%v", !targetInfo.TempDir.DisableCaching),
			fmt.Sprintf("--max-connections=%v", repository.Spec.Backend.MaxConnections()),
			"--metrics-enabled=true",
			"--pushgateway-url=" + pushgateway.URL(),
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
	initContainer.VolumeMounts = UpsertTmpVolumeMount(initContainer.VolumeMounts)

	// mount the volumes specified in RestoreSession inside this init-container
	for _, srcVol := range targetInfo.Target.VolumeMounts {
		initContainer.VolumeMounts = append(initContainer.VolumeMounts, core.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
		})
	}

	// if Repository uses local volume as backend, we have to mount it inside the initContainer
	if repository.Spec.Backend.Local != nil {
		_, mnt := repository.Spec.Backend.Local.ToVolumeAndMount(repository.Name)
		initContainer.VolumeMounts = append(initContainer.VolumeMounts, mnt)
	}

	// pass container runtime settings from RestoreSession to init-container
	if targetInfo.RuntimeSettings.Container != nil {
		initContainer = ofst_util.ApplyContainerRuntimeSettings(initContainer, *targetInfo.RuntimeSettings.Container)
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
		initContainer.SecurityContext = UpsertSecurityContext(securityContext, targetInfo.RuntimeSettings.Container.SecurityContext)
	} else {
		initContainer.SecurityContext = securityContext
	}

	return initContainer
}
