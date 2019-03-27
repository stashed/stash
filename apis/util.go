package apis

var (
	EnableStatusSubresource bool
)

const (
	ModificationTypeInitContainerInjection = "InitContainerInjection"
	ModificationTypeInitContainerDeletion  = "InitContainerDeletion"
	ModificationTypeSidecarInjection       = "SidecarInjection"
	ModificationTypeSidecarDeletion        = "SidecarDeletion"
)

const (
	StashKey   = "stash.appscode.com"
	VersionTag = StashKey + "/tag"
)

const (
	KindDeployment            = "Deployment"
	KindReplicaSet            = "ReplicaSet"
	KindReplicationController = "ReplicationController"
	KindStatefulSet           = "StatefulSet"
	KindDaemonSet             = "DaemonSet"
	KindPersistentVolumeClaim = "PersistentVolumeClaim"
	KindAppBinding            = "AppBinding"
)
