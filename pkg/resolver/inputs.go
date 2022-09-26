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

package resolver

import (
	"fmt"
	"strconv"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	api_util "stash.appscode.dev/apimachinery/pkg/util"
	"stash.appscode.dev/stash/pkg/util"

	core "k8s.io/api/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	appcat "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

func (r *TaskOptions) setVariables() error {
	if r.Variables == nil {
		r.Variables = make(map[string]string)
	}

	if r.Repository != nil {
		err := r.setRepositoryVariables()
		if err != nil {
			return fmt.Errorf("failed to populate variables for Repository %s/%s, Reason: %v",
				r.Repository.Namespace,
				r.Repository.Name,
				err,
			)
		}
	}
	if r.Backup != nil {
		err := r.setBackupInvokerVariables()
		if err != nil {
			return fmt.Errorf("cannot resolve implicit inputs for backup invoker  %s %s/%s, Reason: %v",
				r.Backup.Invoker.GetTypeMeta().Kind,
				r.Backup.Invoker.GetObjectMeta().Namespace,
				r.Backup.Invoker.GetObjectMeta().Name,
				err,
			)
		}
		r.setRetentionPolicyVariables()
	}
	if r.Restore != nil {
		r.setRestoreInvokerVariables()
	}
	r.setSessionVariables()
	r.setInvokerVariables(r.invoker)
	r.setImageVariables()
	r.setLicenseVariables()
	r.setMetricsVariables()

	addon, err := api_util.ExtractAddonInfo(r.CatalogClient, r.task, r.targetRef)
	if err != nil {
		return err
	}
	r.setVariablesFromTaskParams(addon)

	return nil
}

func (r *TaskOptions) setBackupInvokerVariables() error {
	r.setBackupTargetVariables(r.Backup.TargetInfo.Target)

	vars := make(map[string]string)
	// get host name for target
	host, err := util.GetHostName(r.Backup.TargetInfo.Target)
	if err != nil {
		return err
	}
	vars[apis.Hostname] = host

	vars[apis.EnableCache] = strconv.FormatBool(!r.Backup.TargetInfo.TempDir.DisableCaching)

	r.setInterimVolumeVariables(r.Backup.TargetInfo.InterimVolumeTemplate)

	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
	return nil
}

func (r *TaskOptions) setRestoreInvokerVariables() {
	targetInfo := r.Restore.TargetInfo

	vars := make(map[string]string)
	r.setRestoreTargetVariables(targetInfo.Target)

	// append vars from RestoreOptions
	restoreOptions := util.RestoreOptionsForHost(targetInfo.Target.Alias, targetInfo.Target.Rules)
	vars[apis.Hostname] = restoreOptions.Host
	vars[apis.SourceHostname] = restoreOptions.SourceHost
	vars[apis.RestorePaths] = strings.Join(restoreOptions.RestorePaths, ",")
	vars[apis.RestoreSnapshots] = strings.Join(restoreOptions.Snapshots, ",")
	vars[apis.IncludePatterns] = strings.Join(restoreOptions.Include, ",")
	vars[apis.ExcludePatterns] = strings.Join(restoreOptions.Exclude, ",")

	// always enable cache if nothing specified
	vars[apis.EnableCache] = strconv.FormatBool(!targetInfo.TempDir.DisableCaching)

	// pass replicas field to function. if not set pass default 1.
	replicas := int32(1)
	if targetInfo.Target != nil && targetInfo.Target.Replicas != nil {
		replicas = *targetInfo.Target.Replicas
	}
	vars[apis.TargetAppReplicas] = fmt.Sprintf("%d", replicas)

	// interim data volume input
	r.setInterimVolumeVariables(targetInfo.InterimVolumeTemplate)

	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setInvokerVariables(inv invoker.MetadataHandler) {
	vars := map[string]string{
		apis.InvokerKind: inv.GetTypeMeta().Kind,
		apis.InvokerName: inv.GetObjectMeta().Name,
	}
	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setInterimVolumeVariables(interimVolumeTemplate *ofst.PersistentVolumeClaim) {
	vars := make(map[string]string)
	if interimVolumeTemplate != nil {
		vars[apis.InterimDataDir] = apis.StashInterimDataDir
	} else {
		// if interim volume is not specified then use temp dir to store data temporarily
		vars[apis.InterimDataDir] = fmt.Sprintf("%s/stash-interim-volume/data", apis.TmpDirMountPath)
	}
	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setRepositoryVariables() error {
	vars := make(map[string]string)
	if r.Repository == nil {
		return nil
	}

	vars[apis.RepositoryName] = r.Repository.Name
	vars[apis.RepositoryNamespace] = r.Repository.Namespace

	var err error
	if vars[apis.RepositoryProvider], err = r.Repository.Spec.Backend.Provider(); err != nil {
		return err
	}
	if vars[apis.RepositoryBucket], err = r.Repository.Spec.Backend.Container(); err != nil {
		return err
	}
	if vars[apis.RepositoryPrefix], err = r.Repository.Spec.Backend.Prefix(); err != nil {
		return err
	}
	if r.Repository.Spec.Backend.StorageSecretName != "" {
		vars[apis.RepositorySecretName] = r.Repository.Spec.Backend.StorageSecretName
		vars[apis.RepositorySecretNamespace] = r.Repository.Namespace
	}
	if r.Repository.Spec.Backend.S3 != nil && r.Repository.Spec.Backend.S3.Endpoint != "" {
		vars[apis.RepositoryEndpoint] = r.Repository.Spec.Backend.S3.Endpoint
	}
	endpoint, found := r.Repository.Spec.Backend.Endpoint()
	if found {
		vars[apis.RepositoryEndpoint] = endpoint
	}
	region, found := r.Repository.Spec.Backend.Region()
	if found {
		vars[apis.RepositoryRegion] = region
	}
	vars[apis.MaxConnections] = strconv.FormatInt(r.Repository.Spec.Backend.MaxConnections(), 10)

	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
	return nil
}

func (r *TaskOptions) setBackupTargetVariables(target *v1beta1.BackupTarget) {
	vars := make(map[string]string)
	if target != nil {
		r.setTargetVariables(target.Ref)
		if len(target.Paths) > 0 {
			vars[apis.TargetPaths] = strings.Join(target.Paths, ",")
		} else {
			vars[apis.TargetPaths] = apis.StashDefaultMountPath
		}
		r.setDriverVariables()
		r.setMountVariables(target.VolumeMounts)
	}
	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setRestoreTargetVariables(target *v1beta1.RestoreTarget) {
	vars := make(map[string]string)
	if target != nil {
		r.setDriverVariables()
		r.setTargetVariables(target.Ref)
		r.setMountVariables(target.VolumeMounts)
		if len(target.VolumeClaimTemplates) > 0 && target.Ref.Name == "" {
			vars[apis.TargetName] = target.VolumeClaimTemplates[0].Name
		}
	}
	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setMountVariables(mounts []core.VolumeMount) {
	vars := make(map[string]string)
	if len(mounts) > 0 {
		vars[apis.TargetMountPath] = mounts[0].MountPath // We assume that user will provide only one mountPath for stand-alone PVC.
	} else {
		vars[apis.TargetMountPath] = apis.StashDefaultMountPath
	}
	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setDriverVariables() {
	vars := make(map[string]string)

	if len(r.driver.args) > 0 {
		vars[apis.DriverArgs] = strings.Join(r.driver.args, " ")
	}
	if len(r.driver.excludePatterns) > 0 {
		vars[apis.ExcludePatterns] = strings.Join(r.driver.excludePatterns, ",")
	}
	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setTargetVariables(ref v1beta1.TargetRef) {
	vars := make(map[string]string)

	if ref.Kind != "" {
		vars[apis.TargetKind] = ref.Kind
	}
	if ref.Name != "" {
		vars[apis.TargetName] = ref.Name
	}
	if ref.Namespace != "" {
		vars[apis.TargetNamespace] = ref.Namespace
	}

	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setRetentionPolicyVariables() {
	vars := make(map[string]string)

	rp := r.Backup.Invoker.GetRetentionPolicy()
	if rp.KeepLast > 0 {
		vars[apis.RetentionKeepLast] = strconv.FormatInt(rp.KeepLast, 10)
	}
	if rp.KeepHourly > 0 {
		vars[apis.RetentionKeepHourly] = strconv.FormatInt(rp.KeepHourly, 10)
	}
	if rp.KeepDaily > 0 {
		vars[apis.RetentionKeepDaily] = strconv.FormatInt(rp.KeepDaily, 10)
	}
	if rp.KeepWeekly > 0 {
		vars[apis.RetentionKeepWeekly] = strconv.FormatInt(rp.KeepWeekly, 10)
	}
	if rp.KeepMonthly > 0 {
		vars[apis.RetentionKeepMonthly] = strconv.FormatInt(rp.KeepMonthly, 10)
	}
	if rp.KeepYearly > 0 {
		vars[apis.RetentionKeepYearly] = strconv.FormatInt(rp.KeepYearly, 10)
	}
	if len(rp.KeepTags) > 0 {
		vars[apis.RetentionKeepTags] = strings.Join(rp.KeepTags, ",")
	}
	if rp.Prune {
		vars[apis.RetentionPrune] = "true"
	}
	if rp.DryRun {
		vars[apis.RetentionDryRun] = "true"
	}

	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setMetricsVariables() {
	vars := map[string]string{
		apis.PushgatewayURL:    metrics.GetPushgatewayURL(),
		apis.PrometheusJobName: r.invoker.GetObjectMeta().Name,
	}

	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setVariablesFromTaskParams(addon *appcat.StashTaskSpec) {
	if addon == nil {
		return
	}
	vars := make(map[string]string)

	if r.Backup != nil && addon.BackupTask.Name != "" {
		for _, param := range addon.BackupTask.Params {
			vars[param.Name] = param.Value
		}
	}
	if r.Restore != nil && addon.RestoreTask.Name != "" {
		for _, param := range addon.BackupTask.Params {
			vars[param.Name] = param.Value
		}
	}

	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setImageVariables() {
	vars := make(map[string]string)
	vars[apis.StashDockerRegistry] = r.Image.Registry
	vars[apis.StashDockerImage] = r.Image.Image
	vars[apis.StashImageTag] = r.Image.Tag

	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setLicenseVariables() {
	vars := make(map[string]string)
	vars[apis.LicenseApiService] = r.LicenseApiService

	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}

func (r *TaskOptions) setSessionVariables() {
	vars := make(map[string]string)

	if r.Backup != nil {
		vars[apis.BackupSession] = r.Backup.Session.GetObjectMeta().Name
		vars[apis.Namespace] = r.Backup.Session.GetObjectMeta().Namespace
	}
	if r.Restore != nil {
		vars[apis.Namespace] = r.Restore.Invoker.GetObjectMeta().Namespace
		vars[apis.RestoreSession] = r.Restore.Invoker.GetObjectMeta().Name
	}
	r.Variables = meta_util.OverwriteKeys(r.Variables, vars)
}
