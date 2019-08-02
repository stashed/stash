package controller

import (
	"fmt"
	"strconv"
	"strings"

	core_util "kmodules.xyz/client-go/core/v1"
	"stash.appscode.dev/stash/apis"
	apiAlpha "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api "stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
)

func (c *StashController) inputsForBackupConfig(backupConfig api.BackupConfiguration) (map[string]string, error) {
	// get inputs for target
	inputs := c.inputsForBackupTarget(backupConfig.Spec.Target)
	// append inputs for RetentionPolicy
	inputs = core_util.UpsertMap(inputs, c.inputsForRetentionPolicy(backupConfig.Spec.RetentionPolicy))

	// get host name for target
	host, err := util.GetHostName(backupConfig.Spec.Target)
	if err != nil {
		return nil, err
	}
	inputs[apis.Hostname] = host

	// always enable cache if nothing specified
	inputs[apis.EnableCache] = strconv.FormatBool(!backupConfig.Spec.TempDir.DisableCaching)

	// add PushgatewayURL as input
	metricInputs := c.inputForMetrics()
	inputs = core_util.UpsertMap(inputs, metricInputs)

	return inputs, nil
}

func (c *StashController) inputsForRestoreSession(restoreSession api.RestoreSession, host string) (map[string]string, error) {
	// get inputs for target
	inputs := c.inputsForRestoreTarget(restoreSession.Spec.Target)

	// append inputs from RestoreOptions
	restoreOptions := util.RestoreOptionsForHost(host, restoreSession.Spec.Rules)
	inputs[apis.Hostname] = restoreOptions.Host
	inputs[apis.SourceHostname] = restoreOptions.SourceHost
	inputs[apis.RestorePaths] = strings.Join(restoreOptions.RestorePaths, ",")
	inputs[apis.RestoreSnapshots] = strings.Join(restoreOptions.Snapshots, ",")

	// always enable cache if nothing specified
	inputs[apis.EnableCache] = strconv.FormatBool(!restoreSession.Spec.TempDir.DisableCaching)

	// pass replicas field to function. if not set pass default 1.
	replicas := int32(1)
	if restoreSession.Spec.Target != nil && restoreSession.Spec.Target.Replicas != nil {
		replicas = *restoreSession.Spec.Target.Replicas
	}
	inputs[apis.TargetAppReplicas] = fmt.Sprintf("%d", replicas)

	// add PushgatewayURL as input
	metricInputs := c.inputForMetrics()
	inputs = core_util.UpsertMap(inputs, metricInputs)

	return inputs, nil
}

func (c *StashController) inputsForRepository(repository *apiAlpha.Repository) (inputs map[string]string, err error) {
	inputs = make(map[string]string)
	if repository == nil {
		return
	}
	if repository.Name != "" {
		inputs[apis.RepositoryName] = repository.Name
	}
	if inputs[apis.RepositoryProvider], err = util.GetProvider(repository.Spec.Backend); err != nil {
		return
	}
	if inputs[apis.RepositoryBucket], inputs[apis.RepositoryPrefix], err = util.GetBucketAndPrefix(&repository.Spec.Backend); err != nil {
		return
	}
	if repository.Spec.Backend.StorageSecretName != "" {
		inputs[apis.RepositorySecretName] = repository.Spec.Backend.StorageSecretName
	}
	if repository.Spec.Backend.S3 != nil && repository.Spec.Backend.S3.Endpoint != "" {
		inputs[apis.RepositoryEndpoint] = repository.Spec.Backend.S3.Endpoint
	}
	if repository.Spec.Backend.Rest != nil && repository.Spec.Backend.Rest.URL != "" {
		inputs[apis.RepositoryURL] = repository.Spec.Backend.Rest.URL
	}
	inputs[apis.MaxConnections] = strconv.Itoa(util.GetMaxConnections(repository.Spec.Backend))
	return
}

func (c *StashController) inputsForBackupTarget(target *api.BackupTarget) map[string]string {
	inputs := make(map[string]string)
	if target != nil {
		if target.Ref.Name != "" {
			inputs[apis.TargetName] = target.Ref.Name
		}
		if len(target.Paths) > 0 {
			inputs[apis.TargetPaths] = strings.Join(target.Paths, ",")
		}
		if target.VolumeMounts != nil {
			inputs[apis.TargetMountPath] = target.VolumeMounts[0].MountPath
		}
	}
	return inputs
}

func (c *StashController) inputsForRestoreTarget(target *api.RestoreTarget) map[string]string {
	inputs := make(map[string]string)
	if target != nil {
		if target.Ref.Name != "" {
			inputs[apis.TargetName] = target.Ref.Name
		}
		if target.VolumeMounts != nil {
			inputs[apis.TargetMountPath] = target.VolumeMounts[0].MountPath
		}
	}
	return inputs
}

func (c *StashController) inputsForRetentionPolicy(retentionPolicy apiAlpha.RetentionPolicy) map[string]string {
	inputs := make(map[string]string)

	if retentionPolicy.KeepLast > 0 {
		inputs[apis.RetentionKeepLast] = strconv.Itoa(retentionPolicy.KeepLast)
	}
	if retentionPolicy.KeepHourly > 0 {
		inputs[apis.RetentionKeepHourly] = strconv.Itoa(retentionPolicy.KeepHourly)
	}
	if retentionPolicy.KeepDaily > 0 {
		inputs[apis.RetentionKeepDaily] = strconv.Itoa(retentionPolicy.KeepDaily)
	}
	if retentionPolicy.KeepWeekly > 0 {
		inputs[apis.RetentionKeepWeekly] = strconv.Itoa(retentionPolicy.KeepWeekly)
	}
	if retentionPolicy.KeepMonthly > 0 {
		inputs[apis.RetentionKeepMonthly] = strconv.Itoa(retentionPolicy.KeepMonthly)
	}
	if retentionPolicy.KeepYearly > 0 {
		inputs[apis.RetentionKeepYearly] = strconv.Itoa(retentionPolicy.KeepYearly)
	}
	if len(retentionPolicy.KeepTags) > 0 {
		inputs[apis.RetentionKeepTags] = strings.Join(retentionPolicy.KeepTags, ",")
	}
	if retentionPolicy.Prune {
		inputs[apis.RetentionPrune] = "true"
	}
	if retentionPolicy.DryRun {
		inputs[apis.RetentionDryRun] = "true"
	}
	return inputs
}

func (c *StashController) inputForMetrics() map[string]string {
	return map[string]string{
		apis.PushgatewayURL: util.PushgatewayURL(),
	}
}
