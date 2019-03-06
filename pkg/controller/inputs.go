package controller

import (
	"strconv"
	"strings"

	apiAlpha "github.com/appscode/stash/apis/stash/v1alpha1"
	api "github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

// TODO: complete
const (
	RepositoryProvider   = "REPOSITORY_PROVIDER"
	RepositorySecretName = "REPOSITORY_SECRET_NAME"
	RepositoryBucket     = "REPOSITORY_BUCKET"
	RepositoryPrefix     = "REPOSITORY_PREFIX"
	RepositoryEndpoint   = "REPOSITORY_ENDPOINT"

	Hostname = "HOSTNAME"

	TargetName        = "TARGET_NAME"
	TargetDirectories = "TARGET_DIRECTORIES"
	TargetMountPath   = "TARGET_MOUNT_PATH"

	RestoreDirectories = "RESTORE_DIRECTORIES"
	RestoreSnapshots   = "RESTORE_SNAPSHOTS"

	RetentionKeepLast    = "RETENTION_KEEP_LAST"
	RetentionKeepHourly  = "RETENTION_KEEP_HOURLY"
	RetentionKeepDaily   = "RETENTION_KEEP_DAILY"
	RetentionKeepWeekly  = "RETENTION_KEEP_WEEKLY"
	RetentionKeepMonthly = "RETENTION_KEEP_MONTHLY"
	RetentionKeepYearly  = "RETENTION_KEEP_YEARLY"
	RetentionKeepTags    = "RETENTION_KEEP_TAGS"
	RetentionPrune       = "RETENTION_PRUNE"
	RetentionDryRun      = "RETENTION_DRY_RUN"
)

func (c *StashController) inputsForBackupConfig(backupConfig api.BackupConfiguration) (map[string]string, error) {
	// get repository for backupConfig
	repository, err := c.stashClient.StashV1alpha1().Repositories(backupConfig.Namespace).Get(
		backupConfig.Spec.Repository.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}
	// get inputs for repository
	inputs, err := c.inputsForRepository(repository)
	if err != nil {
		return nil, err
	}
	// append inputs for target
	inputs = core_util.UpsertMap(inputs, c.inputsForTarget(backupConfig.Spec.Target))
	// append inputs for RetentionPolicy
	inputs = core_util.UpsertMap(inputs, c.inputsForRetentionPolicy(backupConfig.Spec.RetentionPolicy))
	return inputs, nil
}

func (c *StashController) inputsForRestoreSession(restoreSession api.RestoreSession, host string) (map[string]string, error) {
	// get repository for restoreSession
	repository, err := c.stashClient.StashV1alpha1().Repositories(restoreSession.Namespace).Get(
		restoreSession.Spec.Repository.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}
	// get inputs for repository
	inputs, err := c.inputsForRepository(repository)
	if err != nil {
		return nil, err
	}
	// append inputs for target
	inputs = core_util.UpsertMap(inputs, c.inputsForTarget(restoreSession.Spec.Target))
	// append inputs from RestoreOptions
	restoreOptions := util.RestoreOptionsForHost(host, restoreSession.Spec.Rules)
	inputs[Hostname] = restoreOptions.Host
	inputs[RestoreDirectories] = strings.Join(restoreOptions.RestoreDirs, ",")
	inputs[RestoreSnapshots] = strings.Join(restoreOptions.Snapshots, ",")

	return inputs, nil
}

func (c *StashController) inputsForRepository(repository *apiAlpha.Repository) (inputs map[string]string, err error) {
	inputs = make(map[string]string)
	if repository == nil {
		return
	}
	if inputs[RepositoryProvider], err = util.GetProvider(repository.Spec.Backend); err != nil {
		return
	}
	if inputs[RepositoryBucket], inputs[RepositoryPrefix], err = util.GetBucketAndPrefix(&repository.Spec.Backend); err != nil {
		return
	}
	if repository.Spec.Backend.StorageSecretName != "" {
		inputs[RepositorySecretName] = repository.Spec.Backend.StorageSecretName
	}
	if repository.Spec.Backend.S3 != nil && repository.Spec.Backend.S3.Endpoint != "" {
		inputs[RepositoryEndpoint] = repository.Spec.Backend.S3.Endpoint
	}
	return
}

func (c *StashController) inputsForTarget(target *api.Target) map[string]string {
	inputs := make(map[string]string)
	if target != nil {
		if target.Ref.Name != "" {
			inputs[TargetName] = target.Ref.Name
		}
		if len(target.Directories) > 0 {
			inputs[TargetDirectories] = strings.Join(target.Directories, ",")
		}
		if target.MountPath != "" {
			inputs[TargetMountPath] = target.MountPath
		}
	}
	return inputs
}

func (c *StashController) inputsForRetentionPolicy(retentionPolicy apiAlpha.RetentionPolicy) map[string]string {
	inputs := make(map[string]string)

	if retentionPolicy.KeepLast > 0 {
		inputs[RetentionKeepLast] = strconv.Itoa(retentionPolicy.KeepLast)
	}
	if retentionPolicy.KeepHourly > 0 {
		inputs[RetentionKeepHourly] = strconv.Itoa(retentionPolicy.KeepHourly)
	}
	if retentionPolicy.KeepDaily > 0 {
		inputs[RetentionKeepDaily] = strconv.Itoa(retentionPolicy.KeepDaily)
	}
	if retentionPolicy.KeepWeekly > 0 {
		inputs[RetentionKeepWeekly] = strconv.Itoa(retentionPolicy.KeepWeekly)
	}
	if retentionPolicy.KeepMonthly > 0 {
		inputs[RetentionKeepMonthly] = strconv.Itoa(retentionPolicy.KeepMonthly)
	}
	if retentionPolicy.KeepYearly > 0 {
		inputs[RetentionKeepYearly] = strconv.Itoa(retentionPolicy.KeepYearly)
	}
	if len(retentionPolicy.KeepTags) > 0 {
		inputs[RetentionKeepTags] = strings.Join(retentionPolicy.KeepTags, ",")
	}
	if retentionPolicy.Prune {
		inputs[RetentionPrune] = "true"
	}
	if retentionPolicy.DryRun {
		inputs[RetentionDryRun] = "true"
	}
	return inputs
}
