package v1beta1

const (
	VersionTag = BackupKey + "/tag"
	// ResourceVersion will be used to trigger restarts for ReplicaSet and RC pods
	StashKey                       = "stash.appscode.com"
	BackupKey                      = "backup.appscode.com"
	RestoreKey                     = "restore.appscode.com"
	SuffixLastAppliedConfiguration = "/last-applied-configuration"
	SuffixResourceHash             = "/resource-hash"

	KeyBackupConfigurationTemplate = StashKey + "/backup-template"
	KeyTargetDirectories           = StashKey + "/target-directories"
	KeyMountPath                   = StashKey + "/mountpath"
	KeyVolumeMounts                = StashKey + "/volume-mounts"

	KeyLastAppliedRestoreSession      = RestoreKey + SuffixLastAppliedConfiguration
	KeyLastAppliedBackupConfiguration = BackupKey + SuffixLastAppliedConfiguration

	AppliedBackupConfigurationSpecHash = BackupKey + SuffixResourceHash
	AppliedRestoreSessionSpecHash      = RestoreKey + SuffixResourceHash
	ResourceHash                       = BackupKey + SuffixResourceHash
)
