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

	"stash.appscode.dev/stash/apis"
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/docker"

	"github.com/appscode/go/types"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kmodules.xyz/client-go/tools/analytics"
	"kmodules.xyz/client-go/tools/cli"
	"kmodules.xyz/client-go/tools/clientcmd"
	ofst_util "kmodules.xyz/offshoot-api/util"
)

func NewCheckJob(restic *api_v1alpha1.Restic, hostName, smartPrefix string, image docker.Docker) *batch.Job {
	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CheckJobPrefix + restic.Name,
			Namespace: restic.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: api_v1alpha1.SchemeGroupVersion.String(),
					Kind:       api_v1alpha1.ResourceKindRestic,
					Name:       restic.Name,
					UID:        restic.UID,
				},
			},
			Labels: map[string]string{
				"app":               AppLabelStash,
				AnnotationRestic:    restic.Name,
				AnnotationOperation: OperationCheck,
				// ensure that job gets deleted on completion
				apis.KeyDeleteJobOnCompletion: apis.AllowDeletingJobOnCompletion,
			},
		},
		Spec: batch.JobSpec{
			Template: core.PodTemplateSpec{
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  StashContainer,
							Image: image.ToContainerImage(),
							Args: append([]string{
								"check",
								"--restic-name=" + restic.Name,
								"--host-name=" + hostName,
								"--smart-prefix=" + smartPrefix,
								fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
								fmt.Sprintf("--enable-analytics=%v", cli.EnableAnalytics),
							}, cli.LoggerOptions.ToFlags()...),
							Env: []core.EnvVar{
								{
									Name:  analytics.Key,
									Value: cli.AnalyticsClientID,
								},
							},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      ScratchDirVolumeName,
									MountPath: "/tmp",
								},
							},
						},
					},
					ImagePullSecrets: restic.Spec.ImagePullSecrets,
					RestartPolicy:    core.RestartPolicyOnFailure,
					Volumes: []core.Volume{
						{
							Name: ScratchDirVolumeName,
							VolumeSource: core.VolumeSource{
								EmptyDir: &core.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	// local backend
	// user don't need to specify "stash-local" volume, we collect it from restic-spec
	if restic.Spec.Backend.Local != nil {
		vol, mnt := restic.Spec.Backend.Local.ToVolumeAndMount(LocalVolumeName)
		job.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			job.Spec.Template.Spec.Containers[0].VolumeMounts, mnt)
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, vol)
	}

	return job
}

func NewRecoveryJob(stashClient cs.Interface, recovery *api_v1alpha1.Recovery, image docker.Docker) (*batch.Job, error) {
	repository, err := stashClient.StashV1alpha1().Repositories(recovery.Spec.Repository.Namespace).Get(recovery.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repoLabelData, err := ExtractDataFromRepositoryLabel(repository.Labels)
	if err != nil {
		return nil, err
	}
	volumes := make([]core.Volume, 0)
	volumeMounts := make([]core.VolumeMount, 0)
	for i, recVol := range recovery.Spec.RecoveredVolumes {
		vol, mnt := recVol.ToVolumeAndMount(fmt.Sprintf("vol-%d", i))
		volumes = append(volumes, vol)
		volumeMounts = append(volumeMounts, mnt)
	}

	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RecoveryJobPrefix + recovery.Name,
			Namespace: recovery.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: api_v1alpha1.SchemeGroupVersion.String(),
					Kind:       api_v1alpha1.ResourceKindRecovery,
					Name:       recovery.Name,
					UID:        recovery.UID,
				},
			},
			Labels: map[string]string{
				"app":               AppLabelStash,
				AnnotationRecovery:  recovery.Name,
				AnnotationOperation: OperationRecovery,
				// ensure that the job gets deleted on completion
				apis.KeyDeleteJobOnCompletion: apis.AllowDeletingJobOnCompletion,
			},
		},
		Spec: batch.JobSpec{
			Template: core.PodTemplateSpec{
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  StashContainer,
							Image: image.ToContainerImage(),
							Args: append([]string{
								"recover",
								"--recovery-name=" + recovery.Name,
								fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
								fmt.Sprintf("--enable-analytics=%v", cli.EnableAnalytics),
							}, cli.LoggerOptions.ToFlags()...),
							Env: []core.EnvVar{
								{
									Name:  analytics.Key,
									Value: cli.AnalyticsClientID,
								},
							},
							VolumeMounts: append(volumeMounts, core.VolumeMount{
								Name:      ScratchDirVolumeName,
								MountPath: "/tmp",
							}),
						},
					},
					ImagePullSecrets: recovery.Spec.ImagePullSecrets,
					RestartPolicy:    core.RestartPolicyOnFailure,
					Volumes: append(volumes, core.Volume{
						Name: ScratchDirVolumeName,
						VolumeSource: core.VolumeSource{
							EmptyDir: &core.EmptyDirVolumeSource{},
						},
					}),
				},
			},
		},
	}

	if repoLabelData.WorkloadKind == apis.KindDaemonSet {
		job.Spec.Template.Spec.NodeName = repoLabelData.NodeName
	} else {
		job.Spec.Template.Spec.NodeSelector = recovery.Spec.NodeSelector
	}

	// local backend
	if repository.Spec.Backend.Local != nil {
		w := &api_v1alpha1.LocalTypedReference{
			Kind: repoLabelData.WorkloadKind,
			Name: repoLabelData.WorkloadName,
		}
		_, smartPrefix, err := w.HostnamePrefix(repoLabelData.PodName, repoLabelData.NodeName)
		if err != nil {
			return nil, err
		}
		backend := FixBackendPrefix(repository.Spec.Backend.DeepCopy(), smartPrefix)
		vol, mnt := backend.Local.ToVolumeAndMount(LocalVolumeName)
		job.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			job.Spec.Template.Spec.Containers[0].VolumeMounts, mnt)
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, vol)
	}

	return job, nil
}

// NewPVCRestorerJob return a job definition to restore pvc.
func NewPVCRestorerJob(rs *api_v1beta1.RestoreSession, repository *api_v1alpha1.Repository, image docker.Docker) (*core.PodTemplateSpec, error) {
	container := core.Container{
		Name:  StashContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"restore",
			"--restore-session=" + rs.Name,
			"--restore-model=job",
			"--secret-dir=" + StashSecretMountDir,
			fmt.Sprintf("--enable-cache=%v", !rs.Spec.TempDir.DisableCaching),
			fmt.Sprintf("--max-connections=%v", repository.Spec.Backend.MaxConnections()),
			"--metrics-enabled=true",
			"--pushgateway-url=" + PushgatewayURL(),
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
				Name:      StashSecretVolume,
				MountPath: StashSecretMountDir,
			},
		},
	}

	// mount tmp volume
	container.VolumeMounts = UpsertTmpVolumeMount(container.VolumeMounts)

	// mount the volumes specified in RestoreSession into the job
	for _, srcVol := range rs.Spec.Target.VolumeMounts {
		container.VolumeMounts = append(container.VolumeMounts, core.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
		})
	}

	// if Repository uses local volume as backend, we have to mount it inside the initContainer
	if repository.Spec.Backend.Local != nil {
		_, mnt := repository.Spec.Backend.Local.ToVolumeAndMount(LocalVolumeName)
		container.VolumeMounts = append(container.VolumeMounts, mnt)
	}

	// Pass container RuntimeSettings from RestoreSession
	if rs.Spec.RuntimeSettings.Container != nil {
		container = ofst_util.ApplyContainerRuntimeSettings(container, *rs.Spec.RuntimeSettings.Container)
	}

	// In order to preserve file ownership, restore process need to be run as root user.
	// Stash image uses non-root user "stash"(1005). We have to use securityContext to run stash as root user.
	// If a user specify securityContext either in pod level or container level in RuntimeSetting,
	// don't overwrite that. In this case, user must take the responsibility of possible file ownership modification.
	securityContext := &core.SecurityContext{
		RunAsUser:  types.Int64P(0),
		RunAsGroup: types.Int64P(0),
	}
	if rs.Spec.RuntimeSettings.Container != nil {
		container.SecurityContext = UpsertSecurityContext(securityContext, rs.Spec.RuntimeSettings.Container.SecurityContext)
	} else {
		container.SecurityContext = securityContext
	}

	jobTemplate := &core.PodTemplateSpec{
		Spec: core.PodSpec{
			Containers:    []core.Container{container},
			RestartPolicy: core.RestartPolicyNever,
		},
	}

	// Upsert default pod level security context
	jobTemplate.Spec.SecurityContext = UpsertDefaultPodSecurityContext(jobTemplate.Spec.SecurityContext)

	// Pass pod RuntimeSettings from RestoreSession
	if rs.Spec.RuntimeSettings.Pod != nil {
		jobTemplate.Spec = ofst_util.ApplyPodRuntimeSettings(jobTemplate.Spec, *rs.Spec.RuntimeSettings.Pod)
	}

	// add an emptyDir volume for holding temporary files
	jobTemplate.Spec.Volumes = UpsertTmpVolume(jobTemplate.Spec.Volumes, rs.Spec.TempDir)
	// add storage secret as volume to the workload. this has been mounted on the container above.
	jobTemplate.Spec.Volumes = UpsertSecretVolume(jobTemplate.Spec.Volumes, repository.Spec.Backend.StorageSecretName)
	// if Repository uses local volume as backend, append this volume to the job
	jobTemplate.Spec.Volumes = MergeLocalVolume(jobTemplate.Spec.Volumes, &repository.Spec.Backend)

	return jobTemplate, nil
}

func NewVolumeSnapshotterJob(bs *api_v1beta1.BackupSession, bc *api_v1beta1.BackupConfiguration, image docker.Docker) (*core.PodTemplateSpec, error) {
	container := core.Container{
		Name:  StashContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"create-vs",
			fmt.Sprintf("--backupsession=%s", bs.Name),
			"--metrics-enabled=true",
			"--pushgateway-url=" + PushgatewayURL(),
			fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
			fmt.Sprintf("--enable-analytics=%v", cli.EnableAnalytics),
		}, cli.LoggerOptions.ToFlags()...),
		Env: []core.EnvVar{
			{
				Name: KeyPodName,
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
	}

	// Pass container RuntimeSettings from RestoreSession
	if bc.Spec.RuntimeSettings.Container != nil {
		container = ofst_util.ApplyContainerRuntimeSettings(container, *bc.Spec.RuntimeSettings.Container)
	}

	jobTemplate := &core.PodTemplateSpec{
		Spec: core.PodSpec{
			Containers:    []core.Container{container},
			RestartPolicy: core.RestartPolicyNever,
		},
	}

	// apply default pod level security context
	// don't overwrite user provided sc
	jobTemplate.Spec.SecurityContext = UpsertDefaultPodSecurityContext(jobTemplate.Spec.SecurityContext)

	// Pass pod RuntimeSettings from RestoreSession
	if bc.Spec.RuntimeSettings.Pod != nil {
		jobTemplate.Spec = ofst_util.ApplyPodRuntimeSettings(jobTemplate.Spec, *bc.Spec.RuntimeSettings.Pod)
	}
	return jobTemplate, nil
}

func NewVolumeRestorerJob(rs *api_v1beta1.RestoreSession, image docker.Docker) (*core.PodTemplateSpec, error) {
	container := core.Container{
		Name:  StashContainer,
		Image: image.ToContainerImage(),
		Args: append([]string{
			"restore-vs",
			fmt.Sprintf("--restoresession=%s", rs.Name),
			"--metrics-enabled=true",
			"--pushgateway-url=" + PushgatewayURL(),
			fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
			fmt.Sprintf("--enable-analytics=%v", cli.EnableAnalytics),
		}, cli.LoggerOptions.ToFlags()...),
		Env: []core.EnvVar{
			{
				Name: KeyPodName,
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
	}

	// Pass container RuntimeSettings from RestoreSession
	if rs.Spec.RuntimeSettings.Container != nil {
		container = ofst_util.ApplyContainerRuntimeSettings(container, *rs.Spec.RuntimeSettings.Container)
	}

	jobTemplate := &core.PodTemplateSpec{
		Spec: core.PodSpec{
			Containers:    []core.Container{container},
			RestartPolicy: core.RestartPolicyNever,
		},
	}

	// apply default pod level security context
	// don't overwrite user provided sc
	jobTemplate.Spec.SecurityContext = UpsertDefaultPodSecurityContext(jobTemplate.Spec.SecurityContext)

	// Pass pod RuntimeSettings from RestoreSession
	if rs.Spec.RuntimeSettings.Pod != nil {
		jobTemplate.Spec = ofst_util.ApplyPodRuntimeSettings(jobTemplate.Spec, *rs.Spec.RuntimeSettings.Pod)
	}
	return jobTemplate, nil
}
