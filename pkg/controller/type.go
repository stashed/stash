package controller

import (
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

const (
	BackupConfig     = "backup.appscode.com/config"
	ContainerName    = "restikbackup"
	VolumeName       = "restik-volume"
	Image            = "sauman/restik:test"
	RESTICREPOSITORY = "RESTIC_REPOSITORY"
	BACKUPCRON       = "BACKUP_CRON"
	SOURCEPATH       = "SOURCE_PATH"
	RESTICPASSWORD   = "RESTIC_PASSWORD"
)

type backupServer struct {
	kubeClient *internalclientset.Clientset
}
