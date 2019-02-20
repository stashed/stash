package v1beta1

const (
	BackupKey                = "backup.appscode.com"
	LastAppliedConfiguration = BackupKey + "/last-applied-configuration"
	VersionTag               = BackupKey + "/tag"
	// ResourceVersion will be used to trigger restarts for ReplicaSet and RC pods
	ResourceHash = BackupKey + "/resource-hash"
)
