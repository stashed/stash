package framework

import (
	"fmt"

	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Invocation) BackupTask() v1beta1.Task {
	return v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pvc-backup-task",
		},
		Spec: v1beta1.TaskSpec{
			Steps: []v1beta1.FunctionRef{
				{
					Name: "pvc-backup",
					Params: []v1beta1.Param{
						{
							Name:  outputDir,
							Value: "/tmp/output",
						},
						{
							Name:  tarVol,
							Value: "target-volume",
						},
						{
							Name:  secVol,
							Value: "secret-volume",
						},
					},
				},
				{
					Name: "update-status",
					Params: []v1beta1.Param{
						{
							Name:  outputDir,
							Value: "/tmp/output",
						},
					},
				},
			},
			Volumes: []core.Volume{
				{
					Name: "target-volume",
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("${%s}", apis.TargetName),
						},
					},
				},
				{
					Name: "secret-volume",
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
			Name: "pvc-restore-task",
		},
		Spec: v1beta1.TaskSpec{
			Steps: []v1beta1.FunctionRef{
				{
					Name: "pvc-restore",
					Params: []v1beta1.Param{
						{
							Name:  outputDir,
							Value: "/tmp/output",
						},
						{
							Name:  tarVol,
							Value: "target-volume",
						},
						{
							Name:  secVol,
							Value: "secret-volume",
						},
					},
				},
				{
					Name: "update-status",
					Params: []v1beta1.Param{
						{
							Name:  outputDir,
							Value: "/tmp/output",
						},
					},
				},
			},
			Volumes: []core.Volume{
				{
					Name: "target-volume",
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("${%s}", apis.TargetName),
						},
					},
				},
				{
					Name: "secret-volume",
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
