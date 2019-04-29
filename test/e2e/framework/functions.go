package framework

import (
	"fmt"

	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/docker"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	outputDir            = "outputDir"
	tarVol               = "targetVolume"
	tarVolName           = "target-volume"
	secVol               = "secretVolume"
	secVolName           = "secret-volume"
	RepoSecretMountPath  = "/etc/repository/secret"
	tmpDir               = "/tmp"
	tmpOutputDir         = "/tmp/output"
	PVCBackupTaskName    = "pvc-backup-task"
	PVCRestoreTaskName   = "pvc-restore-task"
	FunctionPvcBackup    = "pvc-backup"
	FunctionPvcRestore   = "pvc-restore"
	FunctionUpdateStatus = "update-status"
)

var (
	DockerRegistry string
	DockerImageTag string
)

func getImage() string {
	return docker.Docker{
		Registry: DockerRegistry,
		Image:    docker.ImageStash,
		Tag:      DockerImageTag,
	}.ToContainerImage()
}

func (f *Invocation) UpdateStatusFunction() v1beta1.Function {
	return v1beta1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name: FunctionUpdateStatus,
		},
		Spec: v1beta1.FunctionSpec{
			Image: getImage(),
			Args: []string{
				FunctionUpdateStatus,
				fmt.Sprintf("--namespace=${%s:=default}", apis.Namespace),
				fmt.Sprintf("--repository=${%s:=}", apis.RepositoryName),
				fmt.Sprintf("--backup-session=${%s:=}", apis.BackupSession),
				fmt.Sprintf("--restore-session=${%s:=}", apis.RestoreSession),
				fmt.Sprintf("--output-dir=${%s:=}", outputDir),
				fmt.Sprintf("--enable-status-subresource=${%s:=false}", apis.StatusSubresourceEnabled),
			},
		},
	}
}

func (f *Invocation) PvcBackupFunction() v1beta1.Function {
	return v1beta1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name: FunctionPvcBackup,
		},
		Spec: v1beta1.FunctionSpec{
			Image: getImage(),
			Args: []string{
				"backup-pvc",
				fmt.Sprintf("--provider=${%s:=}", apis.RepositoryProvider),
				fmt.Sprintf("--bucket=${%s:=}", apis.RepositoryBucket),
				fmt.Sprintf("--endpoint=${%s:=}", apis.RepositoryEndpoint),
				fmt.Sprintf("--path=${%s:=}", apis.RepositoryPrefix),
				fmt.Sprintf("--secret-dir=%s", RepoSecretMountPath),
				fmt.Sprintf("--scratch-dir=%s", tmpDir),
				fmt.Sprintf("--hostname=${%s:=host-0}", apis.Hostname),
				fmt.Sprintf("--backup-dirs=${%s:=}", apis.TargetDirectories),
				fmt.Sprintf("--retention-keep-last=${%s:=0}", apis.RetentionKeepLast),
				fmt.Sprintf("--retention-prune=${%s:=false}", apis.RetentionPrune),
				fmt.Sprintf("--output-dir=${%s:=}", outputDir),
				fmt.Sprintf("--enable-cache=${%s:=true}", apis.EnableCache),
			},
			VolumeMounts: []core.VolumeMount{
				{
					Name:      fmt.Sprintf("${%s}", tarVol),
					MountPath: fmt.Sprintf("${%s}", apis.TargetMountPath),
				},
				{
					Name:      fmt.Sprintf("${%s}", secVol),
					MountPath: RepoSecretMountPath,
				},
			},
		},
	}
}

func (f *Invocation) PvcRestoreFunction() v1beta1.Function {
	return v1beta1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name: FunctionPvcRestore,
		},
		Spec: v1beta1.FunctionSpec{
			Image: getImage(),
			Args: []string{
				"restore-pvc",
				fmt.Sprintf("--provider=${%s:=}", apis.RepositoryProvider),
				fmt.Sprintf("--bucket=${%s:=}", apis.RepositoryBucket),
				fmt.Sprintf("--endpoint=${%s:=}", apis.RepositoryEndpoint),
				fmt.Sprintf("--path=${%s:=}", apis.RepositoryPrefix),
				fmt.Sprintf("--secret-dir=%s", RepoSecretMountPath),
				fmt.Sprintf("--scratch-dir=%s", tmpDir),
				fmt.Sprintf("--hostname=${%s:=host-0}", apis.Hostname),
				fmt.Sprintf("--restore-dirs=${%s:=}", apis.RestoreDirectories),
				fmt.Sprintf("--snapshots=${%s:=}", apis.RestoreSnapshots),
				fmt.Sprintf("--output-dir=${%s:=}", outputDir),
				fmt.Sprintf("--enable-cache=${%s:=true}", apis.EnableCache),
			},
			VolumeMounts: []core.VolumeMount{
				{
					Name:      fmt.Sprintf("${%s}", tarVol),
					MountPath: fmt.Sprintf("${%s}", apis.TargetMountPath),
				},
				{
					Name:      fmt.Sprintf("${%s}", secVol),
					MountPath: RepoSecretMountPath,
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
