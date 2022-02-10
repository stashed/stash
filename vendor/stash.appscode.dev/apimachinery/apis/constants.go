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
	LabelApp            = "app"
	LabelInvokerType    = StashKey + "/invoker-type"
	LabelInvokerName    = StashKey + "/invoker-name"
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
	CallerWebhook    = "webhook"
	CallerController = "controller"
	DefaultHost      = "host-0"
)

// ==================== Prometheus metrics related constants ============
const (
	PromJobStashBackup  = "stash-backup"
	PromJobStashRestore = "stash-restore"

	// RepositoryMetricsPushed whether the Repository metrics for this backup session were pushed or not
	RepositoryMetricsPushed = "RepositoryMetricsPushed"
	// SuccessfullyPushedRepositoryMetrics indicates that the condition transitioned to this state because the repository metrics was successfully pushed to the pushgateway
	SuccessfullyPushedRepositoryMetrics = "SuccessfullyPushedRepositoryMetrics"
	// FailedToPushRepositoryMetrics indicates that the condition transitioned to this state because the Stash was unable to push the repository metrics to the pushgateway
	FailedToPushRepositoryMetrics = "FailedToPushRepositoryMetrics"

	// MetricsPushed whether the metrics for this backup session were pushed or not
	MetricsPushed = "MetricsPushed"
	// SuccessfullyPushedMetrics indicates that the condition transitioned to this state because the metrics was successfully pushed to the pushgateway
	SuccessfullyPushedMetrics = "SuccessfullyPushedMetrics"
	// FailedToPushMetrics indicates that the condition transitioned to this state because the Stash was unable to push the metrics to the pushgateway
	FailedToPushMetrics = "FailedToPushMetrics"
)

// ==================== RBAC related constants ==========================
const (
	KindRole        = "Role"
	KindClusterRole = "ClusterRole"

	StashBackupJobClusterRole            = "stash-backup-job"
	StashRestoreJobClusterRole           = "stash-restore-job"
	StashCronJobClusterRole              = "stash-cron-job"
	StashSidecarClusterRole              = "stash-sidecar"
	StashRestoreInitContainerClusterRole = "stash-restore-init-container"

	StashVolumeSnapshotterClusterRole      = "stash-vs-job"
	StashVolumeSnapshotRestorerClusterRole = "stash-vs-restorer-job"
	StashStorageClassReaderClusterRole     = "stash-sc-reader"
)

// ================== Condition Types Related Constants ===========================
const (
	// RepositoryFound indicates whether the respective Repository object was found or not.
	RepositoryFound = "RepositoryFound"
	// ValidationPassed indicates the validation conditions of the CRD are passed or not.
	ValidationPassed = "ValidationPassed"
	// ResourceValidationPassed indicates that the condition transitioned to this state because the CRD meets validation criteria
	ResourceValidationPassed = "ResourceValidationPassed"
	// ResourceValidationFailed indicates that the condition transitioned to this state because the CRD does not meet validation criteria
	ResourceValidationFailed = "ResourceValidationFailed"
	// BackendSecretFound indicates whether the respective backend secret was found or not.
	BackendSecretFound = "BackendSecretFound"

	// BackupTargetFound indicates whether the backup target was found
	BackupTargetFound = "BackupTargetFound"
	// StashSidecarInjected indicates whether stash sidecar was injected into the targeted workload
	// This condition is applicable only for sidecar model
	StashSidecarInjected = "StashSidecarInjected"
	// CronJobCreated indicates whether the backup triggering CronJob was created
	CronJobCreated = "CronJobCreated"

	// RestoreTargetFound indicates whether the restore target was found
	RestoreTargetFound = "RestoreTargetFound"
	// StashInitContainerInjected indicates whether stash init-container was injected into the targeted workload
	// This condition is applicable only for sidecar model
	StashInitContainerInjected = "StashInitContainerInjected"
	// RestoreJobCreated indicates whether the restore job was created
	RestoreJobCreated = "RestoreJobCreated"
	// RestoreCompleted condition indicates whether the restore process has been completed or not.
	// This condition is particularly helpful when the restore addon require some additional operations to perform
	// before marking the RestoreSession Succeeded/Failed.
	RestoreCompleted = "RestoreCompleted"
	// RestorerEnsured condition indicates whether the restore job / init-container was created or not.
	RestorerEnsured = "RestorerEnsured"

	// GlobalPreBackupHookSucceeded indicates whether the global PreBackupHook was executed successfully or not
	GlobalPreBackupHookSucceeded = "GlobalPreBackupHookSucceeded"
	// GlobalPostBackupHookSucceeded indicates whether the global PostBackupHook was executed successfully or not
	GlobalPostBackupHookSucceeded = "GlobalPostBackupHookSucceeded"
	// GlobalPreRestoreHookSucceeded indicates whether the global PreRestoreHook was executed successfully or not
	GlobalPreRestoreHookSucceeded = "GlobalPreRestoreHookSucceeded"
	// GlobalPostRestoreHookSucceeded indicates whether the global PostRestoreHook was executed successfully or not
	GlobalPostRestoreHookSucceeded = "GlobalPostRestoreHookSucceeded"
	// BackendRepositoryInitialized indicates that whether backend repository was initialized or not
	BackendRepositoryInitialized = "BackendRepositoryInitialized"
	// RetentionPolicyApplied indicates that whether the retention policies were applied or not
	RetentionPolicyApplied = "RetentionPolicyApplied"
	// RepositoryIntegrityVerified indicates whether the repository integrity check succeeded or not
	RepositoryIntegrityVerified = "RepositoryIntegrityVerified"
)

// ================== Condition Types Related Constants ===========================
const (
	// RepositoryAvailable indicates that the condition transitioned to this state because the Repository was available
	RepositoryAvailable = "RepositoryAvailable"
	// RepositoryNotAvailable indicates that the condition transitioned to this state because the Repository was not available
	RepositoryNotAvailable = "RepositoryNotAvailable"
	// UnableToCheckRepositoryAvailability indicates that the condition transitioned to this state because operator was unable
	// to check the Repository availability
	UnableToCheckRepositoryAvailability = "UnableToCheckRepositoryAvailability"

	// BackendSecretAvailable indicates that the condition transitioned to this state because the backend Secret was available
	BackendSecretAvailable = "BackendSecretAvailable"
	// BackendSecretNotAvailable indicates that the condition transitioned to this state because the backend Secret was not available
	BackendSecretNotAvailable = "BackendSecretNotAvailable"
	// UnableToCheckBackendSecretAvailability indicates that the condition transitioned to this state because operator was unable
	// to check the backend Secret availability
	UnableToCheckBackendSecretAvailability = "UnableToCheckBackendSecretAvailability"

	// TargetAvailable indicates that the condition transitioned to this state because the target was available
	TargetAvailable = "TargetAvailable"
	// TargetNotAvailable indicates that the condition transitioned to this state because the target was not available
	TargetNotAvailable = "TargetNotAvailable"
	// UnableToCheckTargetAvailability indicates that the condition transitioned to this state because operator was unable
	// to check the target availability
	UnableToCheckTargetAvailability = "UnableToCheckTargetAvailability"

	// SidecarInjectionSucceeded indicates that the condition transitioned to this state because sidecar was injected
	// successfully into the targeted workload
	SidecarInjectionSucceeded = "SidecarInjectionSucceeded"
	// SidecarInjectionFailed indicates that the condition transitioned to this state because operator was unable
	// to inject sidecar into the targeted workload
	SidecarInjectionFailed = "SidecarInjectionFailed"

	// InitContainerInjectionSucceeded indicates that the condition transitioned to this state because stash init-container
	// was injected successfully into the targeted workload
	InitContainerInjectionSucceeded = "InitContainerInjectionSucceeded"
	// InitContainerInjectionFailed indicates that the condition transitioned to this state because operator was unable
	// to inject stash init-container into the targeted workload
	InitContainerInjectionFailed = "InitContainerInjectionFailed"

	// CronJobCreationSucceeded indicates that the condition transitioned to this state because backup triggering CronJob was created successfully
	CronJobCreationSucceeded = "CronJobCreationSucceeded"
	// CronJobCreationFailed indicates that the condition transitioned to this state because operator was unable to create backup triggering CronJob
	CronJobCreationFailed = "CronJobCreationFailed"

	// RestoreJobCreationSucceeded indicates that the condition transitioned to this state because restore job was created successfully
	RestoreJobCreationSucceeded = "RestoreJobCreationSucceeded"
	// RestoreJobCreationFailed indicates that the condition transitioned to this state because operator was unable to create restore job
	RestoreJobCreationFailed = "RestoreJobCreationFailed"

	// GlobalPreBackupHookExecutedSuccessfully indicates that the condition transitioned to this state because the global PreBackupHook was executed successfully
	GlobalPreBackupHookExecutedSuccessfully = "GlobalPreBackupHookExecutedSuccessfully"
	// GlobalPreBackupHookExecutionFailed indicates that the condition transitioned to this state because the Stash was unable to execute global PreBackupHook
	GlobalPreBackupHookExecutionFailed = "GlobalPreBackupHookExecutionFailed"

	// GlobalPostBackupHookExecutedSuccessfully indicates that the condition transitioned to this state because the global PostBackupHook was executed successfully
	GlobalPostBackupHookExecutedSuccessfully = "GlobalPostBackupHookExecutedSuccessfully"
	// GlobalPostBackupHookExecutionFailed indicates that the condition transitioned to this state because the Stash was unable to execute global PostBackupHook
	GlobalPostBackupHookExecutionFailed = "GlobalPostBackupHookExecutionFailed"

	// GlobalPreRestoreHookExecutedSuccessfully indicates that the condition transitioned to this state because the global PreRestoreHook was executed successfully
	GlobalPreRestoreHookExecutedSuccessfully = "GlobalPreRestoreHookExecutedSuccessfully"
	// GlobalPreRestoreHookExecutionFailed indicates that the condition transitioned to this state because the Stash was unable to execute global PreRestoreHook
	GlobalPreRestoreHookExecutionFailed = "GlobalPreRestoreHookExecutionFailed"

	// GlobalPostRestoreHookExecutedSuccessfully indicates that the condition transitioned to this state because the global PostRestoreHook was executed successfully
	GlobalPostRestoreHookExecutedSuccessfully = "GlobalPostRestoreHookExecutedSuccessfully"
	// GlobalPostRestoreHookExecutionFailed indicates that the condition transitioned to this state because the Stash was unable to execute global PostRestoreHook
	GlobalPostRestoreHookExecutionFailed = "GlobalPostRestoreHookExecutionFailed"

	// BackendRepositoryFound indicates that the condition transitioned to this state because the restic repository was found in the backend
	BackendRepositoryFound = "BackendRepositoryFound"
	// FailedToInitializeBackendRepository indicates that the condition transitioned to this state because the Stash was unable to initialize a repository in the backend
	FailedToInitializeBackendRepository = "FailedToInitializeBackendRepository"
	// SuccessfullyAppliedRetentionPolicy indicates that the condition transitioned to this state because the retention policies was applied successfully
	SuccessfullyAppliedRetentionPolicy = "SuccessfullyAppliedRetentionPolicy"
	// FailedToApplyRetentionPolicy indicates that the condition transitioned to this state because the Stash was unable to apply the retention policies
	FailedToApplyRetentionPolicy = "FailedToApplyRetentionPolicy"
	// SuccessfullyVerifiedRepositoryIntegrity indicates that the condition transitioned to this state because the repository has passed the integrity check
	SuccessfullyVerifiedRepositoryIntegrity = "SuccessfullyVerifiedRepositoryIntegrity"
	// FailedToVerifyRepositoryIntegrity indicates that the condition transitioned to this state because the repository has failed the integrity check
	FailedToVerifyRepositoryIntegrity = "FailedToVerifyRepositoryIntegrity"
)

// ==================== Action related constants ============
const (
	// Pre-backup actions
	InitializeBackendRepository = "InitializeBackendRepository"

	// Post-backup actions
	ApplyRetentionPolicy      = "ApplyRetentionPolicy"
	VerifyRepositoryIntegrity = "VerifyRepositoryIntegrity"
	SendRepositoryMetrics     = "SendRepositoryMetrics"
)
