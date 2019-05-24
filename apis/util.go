package apis

var (
	EnableStatusSubresource bool
)

const (
	StashKey   = "stash.appscode.com"
	VersionTag = StashKey + "/tag"

	KeyDeleteJobOnCompletion = StashKey + "/delete-job-on-completion"
)

const (
	KindDeployment            = "Deployment"
	KindReplicaSet            = "ReplicaSet"
	KindReplicationController = "ReplicationController"
	KindStatefulSet           = "StatefulSet"
	KindDaemonSet             = "DaemonSet"
	KindPersistentVolumeClaim = "PersistentVolumeClaim"
	KindAppBinding            = "AppBinding"
	KindDeploymentConfig      = "DeploymentConfig"
)
