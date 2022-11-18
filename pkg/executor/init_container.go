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

package executor

import (
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	"stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/flags"
	"gomodules.xyz/pointer"
	stringz "gomodules.xyz/x/strings"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/tools/docker"
	ofst_util "kmodules.xyz/offshoot-api/util"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

type InitContainer struct {
	KubeClient        kubernetes.Interface
	OpenshiftClient   oc_cs.Interface
	StashClient       cs.Interface
	RBACOptions       *rbac.Options
	ImagePullSecrets  []core.LocalObjectReference
	Invoker           invoker.RestoreInvoker
	Repository        *v1alpha1.Repository
	LicenseApiService string
	Image             docker.Docker
	Workload          *wapi.Workload
	Caller            string
	Index             int
}

func (e *InitContainer) Ensure() (runtime.Object, kutil.VerbType, error) {
	oldObj := e.Workload.Object.DeepCopyObject()
	sa := stringz.Val(e.Workload.Spec.Template.Spec.ServiceAccountName, "default")
	e.RBACOptions.SetServiceAccountName(sa)

	// Don't create RBAC stuff when the caller is webhook to make the webhooks side effect free.
	if e.Caller != apis.CallerWebhook {
		owner, err := util.OwnerWorkload(e.Workload)
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		e.RBACOptions.SetOwner(owner)

		if err := e.RBACOptions.EnsureRestoreInitContainerRBAC(); err != nil {
			return nil, kutil.VerbUnchanged, err
		}
	}

	e.Workload.Spec.Template.Spec.ImagePullSecrets = e.ImagePullSecrets

	if e.Workload.Spec.Template.Annotations == nil {
		e.Workload.Spec.Template.Annotations = map[string]string{}
	}

	// mark pods with restore invOpts spec hash. used to force restart pods for rc/rs
	e.Workload.Spec.Template.Annotations[api_v1beta1.AppliedRestoreInvokerSpecHash] = e.Invoker.GetHash()

	// insert restore init container
	initContainers := []core.Container{e.newRestoreInitContainer()}
	for i := range e.Workload.Spec.Template.Spec.InitContainers {
		initContainers = core_util.UpsertContainer(initContainers, e.Workload.Spec.Template.Spec.InitContainers[i])
	}
	e.Workload.Spec.Template.Spec.InitContainers = initContainers

	targetInfo := e.Invoker.GetTargetInfo()[e.Index]
	if targetInfo.Target == nil {
		return nil, kutil.VerbUnchanged, fmt.Errorf("target is nil")
	}

	// add an emptyDir volume for holding temporary files
	e.Workload.Spec.Template.Spec.Volumes = util.UpsertTmpVolume(
		e.Workload.Spec.Template.Spec.Volumes,
		targetInfo.TempDir,
	)

	// add RestoreSession definition as annotation of the workload
	if e.Workload.Annotations == nil {
		e.Workload.Annotations = make(map[string]string)
	}
	jsonObj, err := e.Invoker.GetObjectJSON()
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	e.Workload.Annotations[api_v1beta1.KeyLastAppliedRestoreInvoker] = jsonObj
	e.Workload.Annotations[api_v1beta1.KeyLastAppliedRestoreInvokerKind] = e.Invoker.GetTypeMeta().Kind

	// set rolling update so that the workload can automatically update the pods
	setRollingUpdate(e.Workload)

	// apply changes of workload to original object
	if err := wcs.ApplyWorkload(e.Workload.Object, e.Workload); err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	// we don't need to patch the workload when the caller is webhook.
	// the changes will be applied from the webhook automatically.
	if e.Caller == apis.CallerWebhook {
		return nil, kutil.VerbUnchanged, nil
	}
	return ensureWorkloadLatestState(e.KubeClient, e.OpenshiftClient, e.Workload, oldObj)
}

func (e *InitContainer) Cleanup() (runtime.Object, kutil.VerbType, error) {
	oldObj := e.Workload.Object.DeepCopyObject()
	// remove resource hash annotation
	if e.Workload.Spec.Template.Annotations != nil {
		delete(e.Workload.Spec.Template.Annotations, api_v1beta1.AppliedRestoreInvokerSpecHash)
	}
	// remove init-container
	e.Workload.Spec.Template.Spec.InitContainers = core_util.EnsureContainerDeleted(e.Workload.Spec.Template.Spec.InitContainers, apis.StashInitContainer)

	// restore init-container has been removed but workload still may have backup sidecar
	// so removed respective volumes that were added to the workload only if the workload does not have backup sidecar
	if !util.HasStashContainer(e.Workload) {
		// remove the helpers volumes added for init-container
		e.Workload.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(e.Workload.Spec.Template.Spec.Volumes, apis.ScratchDirVolumeName)
	}

	// remove respective annotations
	if e.Workload.Annotations != nil {
		delete(e.Workload.Annotations, api_v1beta1.KeyLastAppliedRestoreInvoker)
		delete(e.Workload.Annotations, api_v1beta1.KeyLastAppliedRestoreInvokerKind)
	}

	// set rolling update so that the workload can automatically update the pods
	setRollingUpdate(e.Workload)

	// apply changes of workload to original object
	if err := wcs.ApplyWorkload(e.Workload.Object, e.Workload); err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	// we don't need to patch the workload when the caller is webhook.
	// the changes will be applied from the webhook automatically.
	if e.Caller == apis.CallerWebhook {
		return nil, kutil.VerbUnchanged, nil
	}
	return ensureWorkloadLatestState(e.KubeClient, e.OpenshiftClient, e.Workload, oldObj)
}

func (e *InitContainer) newRestoreInitContainer() core.Container {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]
	initContainer := core.Container{
		Name:  apis.StashInitContainer,
		Image: e.Image.ToContainerImage(),
		Args: append([]string{
			"restore",
			"--invoker-kind=" + e.Invoker.GetTypeMeta().Kind,
			"--invoker-name=" + e.Invoker.GetObjectMeta().Name,
			"--target-name=" + targetInfo.Target.Ref.Name,
			"--target-namespace=" + targetInfo.Target.Ref.Namespace,
			"--target-kind=" + targetInfo.Target.Ref.Kind,
			fmt.Sprintf("--enable-cache=%v", !targetInfo.TempDir.DisableCaching),
			fmt.Sprintf("--max-connections=%v", e.Repository.Spec.Backend.MaxConnections()),
			"--metrics-enabled=true",
			"--pushgateway-url=" + metrics.GetPushgatewayURL(),
			fmt.Sprintf("--use-kubeapiserver-fqdn-for-aks=%v", clientcmd.UseKubeAPIServerFQDNForAKS()),
		}, flags.LoggerOptions.ToFlags()...),
		Env: []core.EnvVar{
			{
				Name: apis.KeyNodeName,
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			{
				Name: apis.KeyPodName,
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
	}

	// mount tmp volume
	initContainer.VolumeMounts = util.UpsertTmpVolumeMount(initContainer.VolumeMounts)

	// mount the volumes specified in RestoreSession inside this init-container
	for _, srcVol := range targetInfo.Target.VolumeMounts {
		initContainer.VolumeMounts = append(initContainer.VolumeMounts, core.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
		})
	}
	// pass container runtime settings from RestoreSession to init-container
	if targetInfo.RuntimeSettings.Container != nil {
		initContainer = ofst_util.ApplyContainerRuntimeSettings(initContainer, *targetInfo.RuntimeSettings.Container)
	}

	// In order to preserve file ownership, restore process need to be run as root user.
	// Stash image uses non-root user 65535. We have to use securityContext to run stash as root user.
	// If a user specify securityContext either in pod level or container level in RuntimeSetting,
	// don't overwrite that. In this case, user must take the responsibility of possible file ownership modification.
	securityContext := &core.SecurityContext{
		RunAsUser:  pointer.Int64P(0),
		RunAsGroup: pointer.Int64P(0),
	}
	if targetInfo.RuntimeSettings.Container != nil {
		initContainer.SecurityContext = util.UpsertSecurityContext(securityContext, targetInfo.RuntimeSettings.Container.SecurityContext)
	} else {
		initContainer.SecurityContext = securityContext
	}

	return initContainer
}
