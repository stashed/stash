package controller

import (
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

const (
	BackupConfig     = "backup.appscode.com/config"
	ContainerName    = "restikbackup"
	RestickMountPath = "/restik"
	VolumeName       = "restik-volume"
	Image            = "appscode/restik:backup_controller"
	RESTICREPOSITORY = "RESTIC_REPOSITORY"
	BACKUPCRON       = "BACKUP_CRON"
	SOURCEPATH       = "SOURCE_PATH"
	RESTICPASSWORD   = "RESTIC_PASSWORD"
)

type backupServer struct {
	kubeClient *internalclientset.Clientset
}
