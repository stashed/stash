package util

import (
	"fmt"

	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kmodules.xyz/client-go/tools/analytics"
	"kmodules.xyz/client-go/tools/cli"
	"kmodules.xyz/client-go/tools/clientcmd"
	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/docker"
)

func NewCheckJob(restic *api.Restic, hostName, smartPrefix string, image docker.Docker) *batch.Job {
	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CheckJobPrefix + restic.Name,
			Namespace: restic.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: api.SchemeGroupVersion.String(),
					Kind:       api.ResourceKindRestic,
					Name:       restic.Name,
					UID:        restic.UID,
				},
			},
			Labels: map[string]string{
				"app":               AppLabelStash,
				AnnotationRestic:    restic.Name,
				AnnotationOperation: OperationCheck,
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
								fmt.Sprintf("--enable-status-subresource=%v", apis.EnableStatusSubresource),
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

func NewRecoveryJob(stashClient cs.Interface, recovery *api.Recovery, image docker.Docker) (*batch.Job, error) {
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
					APIVersion: api.SchemeGroupVersion.String(),
					Kind:       api.ResourceKindRecovery,
					Name:       recovery.Name,
					UID:        recovery.UID,
				},
			},
			Labels: map[string]string{
				"app":               AppLabelStash,
				AnnotationRecovery:  recovery.Name,
				AnnotationOperation: OperationRecovery,
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
								fmt.Sprintf("--enable-status-subresource=%v", apis.EnableStatusSubresource),
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
		w := &api.LocalTypedReference{
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
