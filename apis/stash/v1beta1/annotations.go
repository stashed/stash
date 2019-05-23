package v1beta1

const (
	// ResourceVersion will be used to trigger restarts for ReplicaSet and RC pods
	StashKey = "stash.appscode.com"

	KeyBackupConfigurationTemplate = StashKey + "/backup-template"
	KeyTargetDirectories           = StashKey + "/target-directories"
	KeyMountPath                   = StashKey + "/mountpath"
	KeyVolumeMounts                = StashKey + "/volume-mounts"

	KeyLastAppliedRestoreSession      = StashKey + "/last-applied-restoresession"
	KeyLastAppliedBackupConfiguration = StashKey + "/last-applied-backupconfiguration"

	AppliedBackupConfigurationSpecHash = StashKey + "/last-applied-backupconfiguration-hash"
	AppliedRestoreSessionSpecHash      = StashKey + "/last-applied-restoresession-hash"
)
