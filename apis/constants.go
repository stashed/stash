/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apis

const (
	StashDockerRegistry = "STASH_DOCKER_REGISTRY"
	StashDockerImage    = "STASH_DOCKER_IMAGE"
	StashImageTag       = "STASH_IMAGE_TAG"
	ImageStash          = "stash"

	Namespace      = "NAMESPACE"
	BackupSession  = "BACKUP_SESSION"
	RestoreSession = "RESTORE_SESSION"

	RepositoryName       = "REPOSITORY_NAME"
	RepositoryProvider   = "REPOSITORY_PROVIDER"
	RepositorySecretName = "REPOSITORY_SECRET_NAME"
	RepositoryBucket     = "REPOSITORY_BUCKET"
	RepositoryPrefix     = "REPOSITORY_PREFIX"
	RepositoryEndpoint   = "REPOSITORY_ENDPOINT"

	Hostname       = "HOSTNAME"
	SourceHostname = "SOURCE_HOSTNAME"
	InterimDataDir = "INTERIM_DATA_DIR"

	TargetName       = "TARGET_NAME"
	TargetAPIVersion = "TARGET_API_VERSION"
	TargetKind       = "TARGET_KIND"
	TargetResource   = "TARGET_RESOURCE"
	TargetNamespace  = "TARGET_NAMESPACE"
	TargetMountPath  = "TARGET_MOUNT_PATH"
	TargetPaths      = "TARGET_PATHS"

	TargetAppVersion  = "TARGET_APP_VERSION"
	TargetAppType     = "TARGET_APP_TYPE"
	TargetAppGroup    = "TARGET_APP_GROUP"
	TargetAppResource = "TARGET_APP_RESOURCE"
	TargetAppReplicas = "TARGET_APP_REPLICAS"

	RestorePaths     = "RESTORE_PATHS"
	RestoreSnapshots = "RESTORE_SNAPSHOTS"

	RetentionKeepLast    = "RETENTION_KEEP_LAST"
	RetentionKeepHourly  = "RETENTION_KEEP_HOURLY"
	RetentionKeepDaily   = "RETENTION_KEEP_DAILY"
	RetentionKeepWeekly  = "RETENTION_KEEP_WEEKLY"
	RetentionKeepMonthly = "RETENTION_KEEP_MONTHLY"
	RetentionKeepYearly  = "RETENTION_KEEP_YEARLY"
	RetentionKeepTags    = "RETENTION_KEEP_TAGS"
	RetentionPrune       = "RETENTION_PRUNE"
	RetentionDryRun      = "RETENTION_DRY_RUN"

	// default true
	// false when TmpDir.DisableCaching is true in backupConfig/restoreSession
	EnableCache    = "ENABLE_CACHE"
	MaxConnections = "MAX_CONNECTIONS"

	// from runtime settings
	NiceAdjustment  = "NICE_ADJUSTMENT"
	IONiceClass     = "IONICE_CLASS"
	IONiceClassData = "IONICE_CLASS_DATA"

	PushgatewayURL    = "PROMETHEUS_PUSHGATEWAY_URL"
	PrometheusJobName = "PROMETHEUS_JOB_NAME"

	StashDefaultVolume          = "stash-volume"
	StashDefaultMountPath       = "/stash-data"
	StashInterimVolume          = "stash-interim-volume"
	StashInterimVolumeMountPath = "/stash-interim-volume"
	StashInterimDataDir         = "/stash-interim-volume/data"

	// backup or restore hooks
	HookType        = "HOOK_TYPE"
	PreBackupHook   = "preBackup"
	PostBackupHook  = "postBackup"
	PreRestoreHook  = "preRestore"
	PostRestoreHook = "postRestore"
	PreTaskHook     = "pre-task-hook"
	PostTaskHook    = "post-task-hook"
)
