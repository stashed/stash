/*
Copyright AppsCode Inc. and Contributors

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

import "time"

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
	KindService               = "Service"
	KindJob                   = "Job"
	KindCronJob               = "CronJob"
	KindNamespace             = "Namespace"
	KindRole                  = "Role"
	KindClusterRole           = "ClusterRole"
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
	ResourcePluralService               = "services"
)

const (
	StashDockerRegistry = "STASH_DOCKER_REGISTRY"
	StashDockerImage    = "STASH_DOCKER_IMAGE"
	StashImageTag       = "STASH_IMAGE_TAG"
	ImageStash          = "stash"

	Namespace      = "NAMESPACE"
	BackupSession  = "BACKUP_SESSION"
	RestoreSession = "RESTORE_SESSION"

	RepositoryName            = "REPOSITORY_NAME"
	RepositoryNamespace       = "REPOSITORY_NAMESPACE"
	RepositoryProvider        = "REPOSITORY_PROVIDER"
	RepositorySecretName      = "REPOSITORY_SECRET_NAME"
	RepositorySecretNamespace = "REPOSITORY_SECRET_NAMESPACE"
	RepositoryBucket          = "REPOSITORY_BUCKET"
	RepositoryPrefix          = "REPOSITORY_PREFIX"
	RepositoryEndpoint        = "REPOSITORY_ENDPOINT"
	RepositoryRegion          = "REPOSITORY_REGION"

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

	InvokerKind = "INVOKER_KIND"
	InvokerName = "INVOKER_NAME"
	AddonImage  = "ADDON_IMAGE"

	ExcludePatterns = "EXCLUDE_PATTERNS"
	IncludePatterns = "INCLUDE_PATTERNS"
	DriverArgs      = "DRIVER_ARGS"

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

	// License related constants
	LicenseApiService = "LICENSE_APISERVICE"
	LicenseReader     = "appscode:license-reader"
)

const (
	PrefixStashBackup         = "stash-backup"
	PrefixStashRestore        = "stash-restore"
	PrefixStashVolumeSnapshot = "stash-vs"
	PrefixStashTrigger        = "stash-trigger"

	StashContainer        = "stash"
	StashInitContainer    = "stash-init"
	StashCronJobContainer = "stash-trigger"
	LocalVolumeName       = "stash-local"
	ScratchDirVolumeName  = "stash-scratchdir"
	TmpDirVolumeName      = "tmp-dir"
	TmpDirMountPath       = "/tmp"
	PodinfoVolumeName     = "stash-podinfo"

	RecoveryJobPrefix   = "stash-recovery-"
	ScaledownCronPrefix = "stash-scaledown-cron-"
	CheckJobPrefix      = "stash-check-"

	AnnotationRestic     = "restic"
	AnnotationRecovery   = "recovery"
	AnnotationOperation  = "operation"
	AnnotationOldReplica = "old-replica"

	OperationRecovery = "recovery"
	OperationCheck    = "check"

	AppLabelStash        = "stash"
	AppLabelStashV1Beta1 = "stash-v1beta1"
	OperationScaleDown   = "scale-down"

	RepositoryFinalizer = "stash"
	SnapshotIDLength    = 8

	ModelSidecar        = "sidecar"
	ModelCronJob        = "cronjob"
	StashSecretVolume   = "stash-secret-volume"
	StashSecretMountDir = "/etc/stash/repository/secret"
	StashNetVolAccessor = "stash-netvol-accessor"

	KeyPodName    = "POD_NAME"
	KeyNodeName   = "NODE_NAME"
	KeyPodOrdinal = "POD_ORDINAL"

	RetryInterval    = 50 * time.Millisecond
	ReadinessTimeout = 2 * time.Minute
)

const (
	LabelApp             = "app"
	LabelInvokerType     = StashKey + "/invoker-type"
	LabelInvokerName     = StashKey + "/invoker-name"
	LabelTargetKind      = StashKey + "/target-kind"
	LabelTargetNamespace = StashKey + "/target-namespace"
	LabelTargetName      = StashKey + "/target-name"
)

const (
	CallerWebhook    = "webhook"
	CallerController = "controller"
	DefaultHost      = "host-0"
)

// ==================== Prometheus metrics related constants ============
const (
	PromJobStashBackup  = "stash-backup"
	PromJobStashRestore = "stash-restore"
)

// ==================== RBAC related constants ==========================
const (
	StashBackupJobClusterRole            = "stash-backup-job"
	StashRestoreJobClusterRole           = "stash-restore-job"
	StashCronJobClusterRole              = "stash-cron-job"
	StashSidecarClusterRole              = "stash-sidecar"
	StashRestoreInitContainerClusterRole = "stash-restore-init-container"

	StashVolumeSnapshotterClusterRole      = "stash-vs-job"
	StashVolumeSnapshotRestorerClusterRole = "stash-vs-restorer-job"
	StashStorageClassReaderClusterRole     = "stash-sc-reader"
)

// =================== Keys for structure logging =====================
const (
	ObjectKey       = "key"
	ObjectKind      = "kind"
	ObjectName      = "name"
	ObjectNamespace = "namespace"

	KeyTargetKind      = "target_kind"
	KeyTargetName      = "target_name"
	KeyTargetNamespace = "target_namespace"

	KeyInvokerKind      = "invoker_kind"
	KeyInvokerName      = "invoker_name"
	KeyInvokerNamespace = "invoker_namespace"

	KeyRepositoryName      = "repo_name"
	KeyRepositoryNamespace = "repo_namespace"

	KeyReason = "reason"
)
