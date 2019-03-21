package util

import (
	"fmt"

	"github.com/appscode/go/types"
	"github.com/appscode/stash/apis"
	v1alpha1_api "github.com/appscode/stash/apis/stash/v1alpha1"
	v1beta1_api "github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/docker"
	core "k8s.io/api/core/v1"
	"kmodules.xyz/client-go/tools/cli"
	"kmodules.xyz/client-go/tools/clientcmd"
)

func NewInitContainer(r *v1alpha1_api.Restic, workload v1alpha1_api.LocalTypedReference, image docker.Docker, enableRBAC bool) core.Container {
	container := NewSidecarContainer(r, workload, image, enableRBAC)
	container.Args = []string{
		"backup",
		"--restic-name=" + r.Name,
		"--workload-kind=" + workload.Kind,
		"--workload-name=" + workload.Name,
		"--docker-registry=" + image.Registry,
		"--image-tag=" + image.Tag,
		"--pushgateway-url=" + PushgatewayURL(),
		fmt.Sprintf("--enable-status-subresource=%v", apis.EnableStatusSubresource),
		fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
		fmt.Sprintf("--enable-analytics=%v", cli.EnableAnalytics),
	}
	container.Args = append(container.Args, cli.LoggerOptions.ToFlags()...)
	if enableRBAC {
		container.Args = append(container.Args, "--enable-rbac=true")
	}

	return container
}

func NewRestoreInitContainer(rs *v1beta1_api.RestoreSession, repository *v1alpha1_api.Repository, image docker.Docker, enableRBAC bool) core.Container {

	initContainer := core.Container{
		Name:  StashInitContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"restore",
			"--restore-session=" + rs.Name,
			"--secret-dir=" + StashSecretMountDir,
			"--metrics-enabled=true",
			"--pushgateway-url=" + PushgatewayURL(),
			fmt.Sprintf("--enable-status-subresource=%v", apis.EnableStatusSubresource),
			fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
			fmt.Sprintf("--enable-analytics=%v", cli.EnableAnalytics),
		}, cli.LoggerOptions.ToFlags()...),
		Env: []core.EnvVar{
			{
				Name: KeyNodeName,
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			{
				Name: KeyPodName,
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      ScratchDirVolumeName,
				MountPath: "/tmp",
			},
			{
				Name:      StashSecretVolume,
				MountPath: StashSecretMountDir,
			},
		},
	}

	// mount the volumes specified in RestoreSession inside this init-container
	for _, srcVol := range rs.Spec.Target.VolumeMounts {
		initContainer.VolumeMounts = append(initContainer.VolumeMounts, core.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
		})
	}

	// if Repository uses local volume as backend, we have to mount it inside the initContainer
	if repository.Spec.Backend.Local != nil {
		_, mnt := repository.Spec.Backend.Local.ToVolumeAndMount(LocalVolumeName)
		initContainer.VolumeMounts = append(initContainer.VolumeMounts, mnt)
	}

	// pass container runtime settings from RestoreSession to init-container
	if rs.Spec.RuntimeSettings.Container != nil {
		// by default container will run as root
		securityContext := &core.SecurityContext{
			RunAsUser:  types.Int64P(0),
			RunAsGroup: types.Int64P(0),
		}
		if rs.Spec.RuntimeSettings.Container.SecurityContext != nil {
			securityContext = rs.Spec.RuntimeSettings.Container.SecurityContext
		}
		initContainer.SecurityContext = securityContext

		initContainer.Resources = rs.Spec.RuntimeSettings.Container.Resources

		if rs.Spec.RuntimeSettings.Container.LivenessProbe != nil {
			initContainer.LivenessProbe = rs.Spec.RuntimeSettings.Container.LivenessProbe
		}
		if rs.Spec.RuntimeSettings.Container.ReadinessProbe != nil {
			initContainer.ReadinessProbe = rs.Spec.RuntimeSettings.Container.ReadinessProbe
		}
		if rs.Spec.RuntimeSettings.Container.Lifecycle != nil {
			initContainer.Lifecycle = rs.Spec.RuntimeSettings.Container.Lifecycle
		}
	}

	return initContainer
}
