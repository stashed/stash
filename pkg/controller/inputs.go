/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"strconv"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	apiAlpha "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/stash/pkg/util"

	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/pushgateway"
)

func (c *StashController) inputsForBackupInvoker(inv invoker.BackupInvoker, targetInfo invoker.BackupTargetInfo) (map[string]string, error) {
	// get inputs for target
	inputs := c.inputsForBackupTarget(targetInfo.Target)
	// append inputs for RetentionPolicy
	inputs = core_util.UpsertMap(inputs, c.inputsForRetentionPolicy(inv.RetentionPolicy))

	// get host name for target
	host, err := util.GetHostName(targetInfo.Target)
	if err != nil {
		return nil, err
	}
	inputs[apis.Hostname] = host

	// invoker information
	inputs[apis.InvokerKind] = inv.TypeMeta.Kind
	inputs[apis.InvokerName] = inv.ObjectMeta.Name

	// always enable cache if nothing specified
	inputs[apis.EnableCache] = strconv.FormatBool(!targetInfo.TempDir.DisableCaching)

	// interim data volume input
	if targetInfo.InterimVolumeTemplate != nil {
		inputs[apis.InterimDataDir] = apis.StashInterimDataDir
	} else {
		// if interim volume is not specified then use temp dir to store data temporarily
		inputs[apis.InterimDataDir] = fmt.Sprintf("%s/stash-interim-volume/data", apis.TmpDirMountPath)
	}

	// add PushgatewayURL as input
	metricInputs := c.inputForMetrics(inv.ObjectMeta.Name)
	inputs = core_util.UpsertMap(inputs, metricInputs)

	return inputs, nil
}

func (c *StashController) inputsForRestoreInvoker(inv invoker.RestoreInvoker, index int) map[string]string {
	targetInfo := inv.TargetsInfo[index]
	// get inputs for target
	inputs := c.inputsForRestoreTarget(targetInfo.Target)

	// append inputs from RestoreOptions
	restoreOptions := util.RestoreOptionsForHost(targetInfo.Target.Alias, targetInfo.Target.Rules)
	inputs[apis.Hostname] = restoreOptions.Host
	inputs[apis.SourceHostname] = restoreOptions.SourceHost
	inputs[apis.RestorePaths] = strings.Join(restoreOptions.RestorePaths, ",")
	inputs[apis.RestoreSnapshots] = strings.Join(restoreOptions.Snapshots, ",")
	inputs[apis.IncludePatterns] = strings.Join(restoreOptions.Include, ",")
	inputs[apis.ExcludePatterns] = strings.Join(restoreOptions.Exclude, ",")

	// invoker information
	inputs[apis.InvokerKind] = inv.TypeMeta.Kind
	inputs[apis.InvokerName] = inv.ObjectMeta.Name

	// always enable cache if nothing specified
	inputs[apis.EnableCache] = strconv.FormatBool(!targetInfo.TempDir.DisableCaching)

	// pass replicas field to function. if not set pass default 1.
	replicas := int32(1)
	if targetInfo.Target != nil && targetInfo.Target.Replicas != nil {
		replicas = *targetInfo.Target.Replicas
	}
	inputs[apis.TargetAppReplicas] = fmt.Sprintf("%d", replicas)

	// interim data volume input
	if targetInfo.InterimVolumeTemplate != nil {
		inputs[apis.InterimDataDir] = apis.StashInterimDataDir
	} else {
		// if interim volume is not specified then use temp dir to store data temporarily
		inputs[apis.InterimDataDir] = fmt.Sprintf("%s/stash-interim-volume/data", apis.TmpDirMountPath)
	}

	// add PushgatewayURL as input
	metricInputs := c.inputForMetrics(inv.ObjectMeta.Name)
	inputs = core_util.UpsertMap(inputs, metricInputs)

	return inputs
}

func (c *StashController) inputsForRepository(repository *apiAlpha.Repository) (inputs map[string]string, err error) {
	inputs = make(map[string]string)
	if repository == nil {
		return
	}
	if repository.Name != "" {
		inputs[apis.RepositoryName] = repository.Name
	}
	if inputs[apis.RepositoryProvider], err = repository.Spec.Backend.Provider(); err != nil {
		return
	}
	if inputs[apis.RepositoryBucket], err = repository.Spec.Backend.Container(); err != nil {
		return
	}
	if inputs[apis.RepositoryPrefix], err = repository.Spec.Backend.Prefix(); err != nil {
		return
	}
	if repository.Spec.Backend.StorageSecretName != "" {
		inputs[apis.RepositorySecretName] = repository.Spec.Backend.StorageSecretName
	}
	if repository.Spec.Backend.S3 != nil && repository.Spec.Backend.S3.Endpoint != "" {
		inputs[apis.RepositoryEndpoint] = repository.Spec.Backend.S3.Endpoint
	}
	endpoint, found := repository.Spec.Backend.Endpoint()
	if found {
		inputs[apis.RepositoryEndpoint] = endpoint
	}
	region, found := repository.Spec.Backend.Region()
	if found {
		inputs[apis.RepositoryRegion] = region
	}
	inputs[apis.MaxConnections] = strconv.FormatInt(repository.Spec.Backend.MaxConnections(), 10)
	return
}

func (c *StashController) inputsForBackupTarget(target *api.BackupTarget) map[string]string {
	inputs := make(map[string]string)
	if target != nil {
		if target.Ref.Name != "" {
			inputs[apis.TargetName] = target.Ref.Name
		}

		if target.Ref.Kind != "" {
			inputs[apis.TargetKind] = target.Ref.Kind
		}
		// If target paths are provided then use them. Otherwise, use stash default mount path.
		if len(target.Paths) > 0 {
			inputs[apis.TargetPaths] = strings.Join(target.Paths, ",")
		} else {
			inputs[apis.TargetPaths] = apis.StashDefaultMountPath
		}
		if len(target.Exclude) > 0 {
			inputs[apis.ExcludePatterns] = strings.Join(target.Exclude, ",")
		}

		// If mount path is provided, then use it. Otherwise, use stash default mount path.
		if len(target.VolumeMounts) > 0 {
			inputs[apis.TargetMountPath] = target.VolumeMounts[0].MountPath // We assume that user will provide only one mountPath for stand-alone PVC.
		} else {
			inputs[apis.TargetMountPath] = apis.StashDefaultMountPath
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
		if target.Ref.Kind != "" {
			inputs[apis.TargetKind] = target.Ref.Kind
		}
		// If mount path is provided, then use it. Otherwise, use stash default mount path.
		if len(target.VolumeMounts) > 0 {
			inputs[apis.TargetMountPath] = target.VolumeMounts[0].MountPath // We assume that user will provide only one mountPath for stand-alone PVC.
		} else {
			inputs[apis.TargetMountPath] = apis.StashDefaultMountPath
		}
		if len(target.VolumeClaimTemplates) > 0 && target.Ref.Name == "" {
			inputs[apis.TargetName] = target.VolumeClaimTemplates[0].Name
		}
	}
	return inputs
}

func (c *StashController) inputsForRetentionPolicy(retentionPolicy apiAlpha.RetentionPolicy) map[string]string {
	inputs := make(map[string]string)

	if retentionPolicy.KeepLast > 0 {
		inputs[apis.RetentionKeepLast] = strconv.FormatInt(retentionPolicy.KeepLast, 10)
	}
	if retentionPolicy.KeepHourly > 0 {
		inputs[apis.RetentionKeepHourly] = strconv.FormatInt(retentionPolicy.KeepHourly, 10)
	}
	if retentionPolicy.KeepDaily > 0 {
		inputs[apis.RetentionKeepDaily] = strconv.FormatInt(retentionPolicy.KeepDaily, 10)
	}
	if retentionPolicy.KeepWeekly > 0 {
		inputs[apis.RetentionKeepWeekly] = strconv.FormatInt(retentionPolicy.KeepWeekly, 10)
	}
	if retentionPolicy.KeepMonthly > 0 {
		inputs[apis.RetentionKeepMonthly] = strconv.FormatInt(retentionPolicy.KeepMonthly, 10)
	}
	if retentionPolicy.KeepYearly > 0 {
		inputs[apis.RetentionKeepYearly] = strconv.FormatInt(retentionPolicy.KeepYearly, 10)
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

func (c *StashController) inputForMetrics(jobName string) map[string]string {
	return map[string]string{
		apis.PushgatewayURL:    pushgateway.URL(),
		apis.PrometheusJobName: jobName,
	}
}
