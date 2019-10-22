package apis

const (
	StashKey   = "stash.appscode.com"
	VersionTag = StashKey + "/tag"

	KeyDeleteJobOnCompletion     = StashKey + "/delete-job-on-completion"
	AllowDeletingJobOnCompletion = "true"
)

const (
	KindDeployment            = "Deployment"
	KindReplicaSet            = "ReplicaSet"
	KindReplicationController = "ReplicationController"
	KindStatefulSet           = "StatefulSet"
	KindDaemonSet             = "DaemonSet"
	KindPod                   = "Pod"
	KindPersistentVolumeClaim = "PersistentVolumeClaim"
	KindAppBinding            = "AppBinding"
	KindDeploymentConfig      = "DeploymentConfig"
	KindSecret                = "Secret"
)

const (
	ResourcePluralDeployment            = "deployments"
	ResourcePluralReplicaSet            = "replicasets"
	ResourcePluralReplicationController = "replicationcontrollers"
	ResourcePluralStatefulSet           = "statefulsets"
	ResourcePluralDaemonSet             = "daemonsets"
	ResourcePluralPod                   = "pods"
	ResourcePluralPersistentVolumeClaim = "persistentvolumeclaims"
	ResourcePluralAppBinding            = "appbindings"
	ResourcePluralDeploymentConfig      = "deploymentconfigs"
	ResourcePluralSecret                = "secrets"
)
