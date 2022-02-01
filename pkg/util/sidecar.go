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
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"

	"gomodules.xyz/flags"
	core "k8s.io/api/core/v1"
	"kmodules.xyz/client-go/tools/clientcmd"
	store "kmodules.xyz/objectstore-api/api/v1"
	ofst_util "kmodules.xyz/offshoot-api/util"
)

func NewBackupSidecarContainer(inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo, backend *store.Backend, image docker.Docker) core.Container {
	sidecar := core.Container{
		Name:  apis.StashContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"run-backup",
			"--invoker-name=" + inv.GetObjectMeta().Name,
			"--invoker-kind=" + inv.GetTypeMeta().Kind,
			"--target-name=" + targetInfo.Target.Ref.Name,
			"--target-kind=" + targetInfo.Target.Ref.Kind,
			fmt.Sprintf("--enable-cache=%v", !targetInfo.TempDir.DisableCaching),
			fmt.Sprintf("--max-connections=%v", backend.MaxConnections()),
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
	sidecar.VolumeMounts = UpsertTmpVolumeMount(sidecar.VolumeMounts)

	// mount the volumes specified in invoker this sidecar
	for _, srcVol := range targetInfo.Target.VolumeMounts {
		sidecar.VolumeMounts = append(sidecar.VolumeMounts, core.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
		})
	}

	// pass container runtime settings from invoker to sidecar
	if targetInfo.RuntimeSettings.Container != nil {
		sidecar = ofst_util.ApplyContainerRuntimeSettings(sidecar, *targetInfo.RuntimeSettings.Container)
	}
	return sidecar
}
