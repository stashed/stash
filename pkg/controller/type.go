package controller

import (
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

const (
	BackupConfig      = "backup.appscode.com/config"
	ContainerName     = "restic-sidecar"
	RESTIC_REPOSITORY = "RESTIC_REPOSITORY"
	BACKUP_CRON       = "BACKUP_CRON"
	SOURCE_PATH       = "SOURCE_PATH"
	RESTIC_PASSWORD   = "RESTIC_PASSWORD"
)

type backupServer struct {
	kubeClient *internalclientset.Clientset
}
