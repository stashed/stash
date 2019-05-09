package framework

import (
	"fmt"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
)

func (f *Invocation) BackupTask() v1beta1.Task {
	return v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name: PVCBackupTaskName,
		},
		Spec: v1beta1.TaskSpec{
			Steps: []v1beta1.FunctionRef{
				{
					Name: FunctionPvcBackup,
					Params: []v1beta1.Param{
						{
							Name:  outputDir,
							Value: tmpOutputDir,
						},
						{
							Name:  tarVol,
							Value: tarVolName,
						},
						{
							Name:  secVol,
							Value: secVolName,
						},
					},
				},
				{
					Name: FunctionUpdateStatus,
					Params: []v1beta1.Param{
						{
							Name:  outputDir,
							Value: tmpOutputDir,
						},
					},
				},
			},
			Volumes: []core.Volume{
				{
					Name: tarVolName,
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("${%s}", apis.TargetName),
						},
					},
				},
				{
					Name: secVolName,
					VolumeSource: core.VolumeSource{
						Secret: &core.SecretVolumeSource{
							SecretName: fmt.Sprintf("${%s}", apis.RepositorySecretName),
						},
					},
				},
			},
		},
	}
}

func (f *Invocation) RestoreTask() v1beta1.Task {
	return v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name: PVCRestoreTaskName,
		},
		Spec: v1beta1.TaskSpec{
			Steps: []v1beta1.FunctionRef{
				{
					Name: FunctionPvcRestore,
					Params: []v1beta1.Param{
						{
							Name:  outputDir,
							Value: tmpOutputDir,
						},
						{
							Name:  tarVol,
							Value: tarVolName,
						},
						{
							Name:  secVol,
							Value: secVolName,
						},
					},
				},
				{
					Name: FunctionUpdateStatus,
					Params: []v1beta1.Param{
						{
							Name:  outputDir,
							Value: tmpOutputDir,
						},
					},
				},
			},
			Volumes: []core.Volume{
				{
					Name: tarVolName,
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("${%s}", apis.TargetName),
						},
					},
				},
				{
					Name: secVolName,
					VolumeSource: core.VolumeSource{
						Secret: &core.SecretVolumeSource{
							SecretName: fmt.Sprintf("${%s}", apis.RepositorySecretName),
						},
					},
				},
			},
		},
	}
}
