package controller

import (
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

const (
	BackupConfig    = "backup.appscode.com/config"
	ContainerName   = "restic-sidecar"
	Namespace       = "RESTIK_NAMESPACE"
	TPR             = "TPR"
	RESTIC_PASSWORD = "RESTIC_PASSWORD"
)

type backupServer struct {
	kubeClient *internalclientset.Clientset
}
