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

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	api_util "stash.appscode.dev/apimachinery/pkg/util"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	appcatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

type TaskOptions struct {
	StashClient       cs.Interface
	CatalogClient     appcatalog_cs.Interface
	Repository        *v1alpha1.Repository
	Image             docker.Docker
	LicenseApiService string
	Variables         map[string]string
	Backup            *BackupOptions
	Restore           *RestoreOptions

	runtimeSettings ofst.RuntimeSettings
	tempDir         v1beta1.EmptyDirSettings
	targetRef       v1beta1.TargetRef
	task            v1beta1.TaskRef
	invoker         invoker.MetadataHandler
	driver          driverOptions
}

type BackupOptions struct {
	TargetInfo invoker.BackupTargetInfo
	Invoker    invoker.BackupInvoker
	Session    *invoker.BackupSessionHandler
}

type RestoreOptions struct {
	TargetInfo invoker.RestoreTargetInfo
	Invoker    invoker.RestoreInvoker
}

type driverOptions struct {
	args            []string
	excludePatterns []string
}

func (r *TaskOptions) Resolve() (core.PodSpec, error) {
	r.setInvokerOptions()

	err := r.setAddonInfo()
	if err != nil {
		return core.PodSpec{}, err
	}

	invRef, err := r.invoker.GetObjectRef()
	if err != nil {
		return core.PodSpec{}, err
	}

	if err := r.setVariables(); err != nil {
		return core.PodSpec{}, nil
	}
	r.setDefaultSecurityContext()

	tr := taskResolver{
		stashClient:     r.StashClient,
		taskName:        r.task.Name,
		inputs:          r.Variables,
		runtimeSettings: r.runtimeSettings,
		tempDir:         r.tempDir,
		inv:             invRef,
		target:          r.targetRef,
	}
	tr.setHookOptions(r)

	podSpec, err := tr.getPodSpec()
	if err != nil {
		return core.PodSpec{}, fmt.Errorf("can't get PodSpec for backup invoker %s/%s, Reason: %v",
			r.invoker.GetObjectMeta().Namespace,
			r.invoker.GetObjectMeta().Name,
			err,
		)
	}

	return podSpec, nil
}

func (r *TaskOptions) setInvokerOptions() {
	if r.Backup != nil {
		r.invoker = r.Backup.Invoker
		r.targetRef = r.Backup.TargetInfo.Target.Ref
		r.task = r.Backup.TargetInfo.Task
		r.driver = driverOptions{
			args:            r.Backup.TargetInfo.Target.Args,
			excludePatterns: r.Backup.TargetInfo.Target.Exclude,
		}
		r.runtimeSettings = r.Backup.TargetInfo.RuntimeSettings
		r.tempDir = r.Backup.TargetInfo.TempDir
	}

	if r.Restore != nil {
		r.invoker = r.Restore.Invoker
		r.targetRef = r.Restore.TargetInfo.Target.Ref
		r.task = r.Restore.TargetInfo.Task
		r.driver = driverOptions{
			args: r.Restore.TargetInfo.Target.Args,
		}
		r.runtimeSettings = r.Restore.TargetInfo.RuntimeSettings
		r.tempDir = r.Restore.TargetInfo.TempDir
	}
}

func (r *TaskOptions) setAddonInfo() error {
	addon, err := api_util.ExtractAddonInfo(r.CatalogClient, r.task, r.targetRef)
	if err != nil {
		return err
	}

	if r.Backup != nil {
		r.task.Name = addon.BackupTask.Name
		r.setTaskParams(addon.BackupTask.Params)
	}
	if r.Restore != nil {
		r.task.Name = addon.RestoreTask.Name
		r.setTaskParams(addon.RestoreTask.Params)
	}
	return nil
}

func (r *TaskOptions) setTaskParams(params []appcatalog.Param) {
	r.task.Params = make([]v1beta1.Param, 0)
	for i := range params {
		r.task.Params = append(r.task.Params, v1beta1.Param{
			Name:  params[i].Name,
			Value: params[i].Value,
		})
	}
}

func (r *TaskOptions) setDefaultSecurityContext() {
	if r.Restore != nil {
		r.setDefaultSecurityContextForRestore()
	}
}

func (r *TaskOptions) setDefaultSecurityContextForRestore() {
	// In order to preserve file ownership, restore process need to be run as root user.
	// Stash image uses non-root user 65535. We have to use securityContext to run stash as root user.
	// If a user specify securityContext either in pod level or container level in RuntimeSetting,
	// don't overwrite that. In this case, user must take the responsibility of possible file ownership modification.
	defaultSecurityContext := &core.PodSecurityContext{
		RunAsUser:  pointer.Int64P(0),
		RunAsGroup: pointer.Int64P(0),
	}
	if r.runtimeSettings.Pod == nil {
		r.runtimeSettings.Pod = &ofst.PodRuntimeSettings{}
	}
	r.runtimeSettings.Pod.SecurityContext = util.UpsertPodSecurityContext(defaultSecurityContext, r.runtimeSettings.Pod.SecurityContext)
}
