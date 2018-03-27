package v1alpha1

const (
	ResticKey                = "restic.appscode.com"
	LastAppliedConfiguration = ResticKey + "/last-applied-configuration"
	VersionTag               = ResticKey + "/tag"
	// ResourceVersion will be used to trigger restarts for ReplicaSet and RC pods
	ResourceHash = ResticKey + "/resource-hash"
)
