package controller

import (
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

const (
	BackupConfig     = "config"
	ContainerName    = "restikbackup"
	RestickMountPath = "/restik"
	VolumeName       = "restik-volume"
	Image            = "sauman/restik:backup_controller"
)

type backupServer struct {
	kubeClient *internalclientset.Clientset
}
