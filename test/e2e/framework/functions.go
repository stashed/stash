package framework

import (
	"fmt"

	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	outputDir = "outputDir"
	tarVol    = "targetVolume"
	secVol    = "secretVolume"
)

func (f *Invocation) UpdateStatusFunction() v1beta1.Function {
	return v1beta1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name: "update-status",
		},
		Spec: v1beta1.FunctionSpec{
			Image: "suaas21/stash:e2e-test-volume",
			Args: []string{
				"update-status",
				fmt.Sprintf("--namespace=${%s:=default}", apis.Namespace),
				fmt.Sprintf("--repository=${%s:=}", apis.RepositoryName),
				fmt.Sprintf("--backup-session=${%s:=}", apis.BackupSession),
				fmt.Sprintf("--restore-session=${%s:=}", apis.RestoreSession),
				fmt.Sprintf("--output-dir=${%s:=}", "outputDir"),
				fmt.Sprintf("--enable-status-subresource=${%s:=%v}", apis.StatusSubresourceEnabled, false),
			},
		},
	}
}

func (f *Invocation) BackupFunction() v1beta1.Function {
	return v1beta1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pvc-backup",
		},
		Spec: v1beta1.FunctionSpec{
			Image: "suaas21/stash:e2e-test-volume",
			Args: []string{
				"backup-pvc",
				fmt.Sprintf("--provider=${%s:=}", apis.RepositoryProvider),
				fmt.Sprintf("--bucket=${%s:=}", apis.RepositoryBucket),
				fmt.Sprintf("--endpoint=${%s:=}", apis.RepositoryEndpoint),
				fmt.Sprintf("--path=${%s:=}", apis.RepositoryPrefix),
				fmt.Sprintf("--secret-dir=%s", "/etc/repository/secret"),
				fmt.Sprintf("--scratch-dir=%s", "/tmp"),
				fmt.Sprintf("--hostname=${%s:=host-0}", apis.Hostname),
				fmt.Sprintf("--backup-dirs=${%s:=}", apis.TargetDirectories),
				fmt.Sprintf("--retention-keep-last=${%s:=0}", apis.RetentionKeepLast),
				fmt.Sprintf("--retention-keep-tags=${%s:=}", apis.RetentionKeepTags),
				fmt.Sprintf("--retention-prune=${%s:=false}", apis.RetentionPrune),
				fmt.Sprintf("--output-dir=${%s:=}", outputDir),
				fmt.Sprintf("--enable-cache=${%s:=true}", apis.EnableCache),
			},
			VolumeMounts: []core.VolumeMount{
				{
					Name:      fmt.Sprintf("${%s}", "targetVolume"),
					MountPath: fmt.Sprintf("${%s}", apis.TargetMountPath),
				},
				{
					Name:      fmt.Sprintf("${%s}", "secretVolume"),
					MountPath: fmt.Sprintf("/etc/repository/secret"),
				},
			},
		},
	}
}

func (f *Invocation) RestoreFunction() v1beta1.Function {
	return v1beta1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pvc-restore",
		},
		Spec: v1beta1.FunctionSpec{
			Image: "suaas21/stash:e2e-test-volume",
			Args: []string{
				"restore-pvc",
				fmt.Sprintf("--provider=${%s:=}", apis.RepositoryProvider),
				fmt.Sprintf("--bucket=${%s:=}", apis.RepositoryBucket),
				fmt.Sprintf("--endpoint=${%s:=}", apis.RepositoryEndpoint),
				fmt.Sprintf("--path=${%s:=}", apis.RepositoryPrefix),
				fmt.Sprintf("--secret-dir=%s", "/etc/repository/secret"),
				fmt.Sprintf("--scratch-dir=%s", "/tmp"),
				fmt.Sprintf("--hostname=${%s:=host-0}", apis.Hostname),
				fmt.Sprintf("--restore-dirs=${%s:=}", apis.RestoreDirectories),
				fmt.Sprintf("--snapshots=${%s:=}", apis.RestoreSnapshots),
				fmt.Sprintf("--output-dir=${%s:=}", outputDir),
				fmt.Sprintf("--enable-cache=${%s:=true}", apis.EnableCache),
			},
			VolumeMounts: []core.VolumeMount{
				{
					Name:      fmt.Sprintf("${%s}", "targetVolume"),
					MountPath: fmt.Sprintf("${%s}", apis.TargetMountPath),
				},
				{
					Name:      fmt.Sprintf("${%s}", "secretVolume"),
					MountPath: fmt.Sprintf("/etc/repository/secret"),
				},
			},
		},
	}
}

func (f *Invocation) CreateFunction(function v1beta1.Function) error {
	_, err := f.StashClient.StashV1beta1().Functions().Create(&function)
	return err
}

func (f *Invocation) CreateTask(task v1beta1.Task) error {
	_, err := f.StashClient.StashV1beta1().Tasks().Create(&task)
	return err
}

func (f *Invocation) DeleteFunction(meta metav1.ObjectMeta) error {
	return f.StashClient.StashV1beta1().Functions().Delete(meta.Name, &metav1.DeleteOptions{})

}

func (f *Invocation) DeleteTask(meta metav1.ObjectMeta) error {
	return f.StashClient.StashV1beta1().Tasks().Delete(meta.Name, &metav1.DeleteOptions{})

}
