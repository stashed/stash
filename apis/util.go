package apis

var (
	EnableStatusSubresource bool
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
	KindDeploymentConfig      = "DeploymentConfig"
)

const (
	APIVersionAppsV1           = "apps/v1"
	APIVersionAppsV1beta1      = "apps/v1beta1"
	APIVersionAppsV1beta2      = "apps/v1beta2"
	APIVersionCoreV1           = "v1"
	APIVersionExtensionV1beta1 = "extensions/v1beta1"
)
