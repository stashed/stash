package controller

import (
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

const (
	BackupConfig     = "backup.appscode.com/config"
	ContainerName    = "restikbackup"
	RestickMountPath = "/restik"
	VolumeName       = "restik-volume"
	Image            = "sauman/restik:0.0.7"
)

type backupServer struct {
	kubeClient *internalclientset.Clientset
}
