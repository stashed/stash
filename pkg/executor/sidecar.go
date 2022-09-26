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
	"context"
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/metrics"
	"stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/flags"
	stringz "gomodules.xyz/x/strings"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
	apps_util "kmodules.xyz/client-go/apps/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/clientcmd"
	ofst_util "kmodules.xyz/offshoot-api/util"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
	ocapps_util "kmodules.xyz/openshift/client/clientset/versioned/typed/apps/v1/util"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

type Sidecar struct {
	KubeClient        kubernetes.Interface
	OpenshiftClient   oc_cs.Interface
	StashClient       cs.Interface
	RBACOptions       *rbac.Options
	ImagePullSecrets  []core.LocalObjectReference
	Invoker           invoker.BackupInvoker
	Repository        *v1alpha1.Repository
	LicenseApiService string
	Image             docker.Docker
	Workload          *wapi.Workload
	Caller            string
	Index             int
}

func (e *Sidecar) Ensure() (runtime.Object, kutil.VerbType, error) {
	oldObj := e.Workload.Object.DeepCopyObject()
	sa := stringz.Val(e.Workload.Spec.Template.Spec.ServiceAccountName, "default")
	e.RBACOptions.SetServiceAccountName(sa)

	if e.Caller != apis.CallerWebhook {
		owner, err := util.OwnerWorkload(e.Workload)
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		e.RBACOptions.SetOwner(owner)

		if err := e.RBACOptions.EnsureSideCarRBAC(); err != nil {
			return nil, kutil.VerbUnchanged, err
		}
	}

	e.Workload.Spec.Template.Spec.ImagePullSecrets = e.ImagePullSecrets

	if e.Workload.Spec.Template.Annotations == nil {
		e.Workload.Spec.Template.Annotations = map[string]string{}
	}
	// mark pods with BackupConfiguration spec hash. used to force restart pods for rc/rs
	e.Workload.Spec.Template.Annotations[api_v1beta1.AppliedBackupInvokerSpecHash] = e.Invoker.GetHash()

	targetInfo := e.Invoker.GetTargetInfo()[e.Index]
	if targetInfo.Target == nil {
		return nil, kutil.VerbUnchanged, fmt.Errorf("target is nil")
	}

	e.Workload.Spec.Template.Spec.Containers = core_util.UpsertContainer(
		e.Workload.Spec.Template.Spec.Containers,
		e.newBackupSidecar(),
	)

	e.Workload.Spec.Template.Spec.Volumes = util.UpsertTmpVolume(e.Workload.Spec.Template.Spec.Volumes, targetInfo.TempDir)

	if e.Workload.Annotations == nil {
		e.Workload.Annotations = make(map[string]string)
	}
	jsonObj, err := e.Invoker.GetObjectJSON()
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	e.Workload.Annotations[api_v1beta1.KeyLastAppliedBackupInvoker] = jsonObj
	e.Workload.Annotations[api_v1beta1.KeyLastAppliedBackupInvokerKind] = e.Invoker.GetTypeMeta().Kind

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

func (e *Sidecar) Cleanup() (runtime.Object, kutil.VerbType, error) {
	oldObj := e.Workload.Object.DeepCopyObject()
	// remove resource hash annotation
	if e.Workload.Spec.Template.Annotations != nil {
		delete(e.Workload.Spec.Template.Annotations, api_v1beta1.AppliedBackupInvokerSpecHash)
	}
	// remove sidecar container
	e.Workload.Spec.Template.Spec.Containers = core_util.EnsureContainerDeleted(e.Workload.Spec.Template.Spec.Containers, apis.StashContainer)

	// backup sidecar has been removed but workload still may have restore init-container
	// so removed respective volumes that were added to the workload only if the workload does not have restore init-container
	if !util.HasStashContainer(e.Workload) {
		// remove the helpers volumes that were added for sidecar
		e.Workload.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(e.Workload.Spec.Template.Spec.Volumes, apis.ScratchDirVolumeName)
	}

	// remove respective annotations
	if e.Workload.Annotations != nil {
		delete(e.Workload.Annotations, api_v1beta1.KeyLastAppliedBackupInvoker)
		delete(e.Workload.Annotations, api_v1beta1.KeyLastAppliedBackupInvokerKind)
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

func (e *Sidecar) newBackupSidecar() core.Container {
	targetInfo := e.Invoker.GetTargetInfo()[e.Index]

	sidecar := core.Container{
		Name:  apis.StashContainer,
		Image: e.Image.ToContainerImage(),
		Args: append([]string{
			"run-backup",
			"--invoker-name=" + e.Invoker.GetObjectMeta().Name,
			"--invoker-kind=" + e.Invoker.GetTypeMeta().Kind,
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
	sidecar.VolumeMounts = util.UpsertTmpVolumeMount(sidecar.VolumeMounts)

	// mount the volumes specified in invoker this sidecar
	for _, srcVol := range targetInfo.Target.VolumeMounts {
		sidecar.VolumeMounts = append(sidecar.VolumeMounts, core.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
		})
	}

	// pass container runtime settings from invoker to sidecar
	if targetInfo.RuntimeSettings.Container != nil {
		sidecar = ofst_util.ApplyContainerRuntimeSettings(sidecar, *targetInfo.RuntimeSettings.Container)
	}
	return sidecar
}

func ensureWorkloadLatestState(
	kubeClient kubernetes.Interface,
	ocClient oc_cs.Interface,
	w *wapi.Workload,
	oldObj runtime.Object,
) (runtime.Object, kutil.VerbType, error) {
	switch w.Kind {
	case apis.KindDeployment:
		updatedObj, verb, err := apps_util.PatchDeploymentObject(context.TODO(), kubeClient, oldObj.(*appsv1.Deployment), w.Object.(*appsv1.Deployment), metav1.PatchOptions{})
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		if verb == kutil.VerbPatched {
			return updatedObj, verb, util.WaitUntilDeploymentReady(kubeClient, oldObj.(*appsv1.Deployment).ObjectMeta)
		}

	case apis.KindDaemonSet:
		updatedObj, verb, err := apps_util.PatchDaemonSetObject(context.TODO(), kubeClient, oldObj.(*appsv1.DaemonSet), w.Object.(*appsv1.DaemonSet), metav1.PatchOptions{})
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		if verb == kutil.VerbPatched {
			return updatedObj, verb, util.WaitUntilDaemonSetReady(kubeClient, oldObj.(*appsv1.DaemonSet).ObjectMeta)
		}
	case apis.KindStatefulSet:
		updatedObj, verb, err := apps_util.PatchStatefulSetObject(context.TODO(), kubeClient, oldObj.(*appsv1.StatefulSet), w.Object.(*appsv1.StatefulSet), metav1.PatchOptions{})
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		if verb == kutil.VerbPatched {
			return updatedObj, verb, util.WaitUntilStatefulSetReady(kubeClient, oldObj.(*appsv1.StatefulSet).ObjectMeta)
		}
	case apis.KindDeploymentConfig:
		updatedObj, verb, err := ocapps_util.PatchDeploymentConfigObject(context.TODO(), ocClient, oldObj.(*ocapps.DeploymentConfig), w.Object.(*ocapps.DeploymentConfig), metav1.PatchOptions{})
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		if verb == kutil.VerbPatched {
			return updatedObj, verb, util.WaitUntilDeploymentConfigReady(ocClient, oldObj.(*ocapps.DeploymentConfig).ObjectMeta)
		}
	default:
		return nil, kutil.VerbUnchanged, fmt.Errorf("unkown workload kind: %s", w.Kind)
	}
	return nil, kutil.VerbUnchanged, nil
}

func setRollingUpdate(w *wapi.Workload) {
	switch t := w.Object.(type) {
	case *extensions.DaemonSet:
		t.Spec.UpdateStrategy.Type = extensions.RollingUpdateDaemonSetStrategyType
		if t.Spec.UpdateStrategy.RollingUpdate == nil ||
			t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable == nil ||
			t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue() == 0 {
			count := intstr.FromInt(1)
			t.Spec.UpdateStrategy.RollingUpdate = &extensions.RollingUpdateDaemonSet{
				MaxUnavailable: &count,
			}
		}
	case *appsv1beta2.DaemonSet:
		t.Spec.UpdateStrategy.Type = appsv1beta2.RollingUpdateDaemonSetStrategyType
		if t.Spec.UpdateStrategy.RollingUpdate == nil ||
			t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable == nil ||
			t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue() == 0 {
			count := intstr.FromInt(1)
			t.Spec.UpdateStrategy.RollingUpdate = &appsv1beta2.RollingUpdateDaemonSet{
				MaxUnavailable: &count,
			}
		}
	case *appsv1.DaemonSet:
		t.Spec.UpdateStrategy.Type = appsv1.RollingUpdateDaemonSetStrategyType
		if t.Spec.UpdateStrategy.RollingUpdate == nil ||
			t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable == nil ||
			t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue() == 0 {
			count := intstr.FromInt(1)
			t.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &count,
			}
		}
	case *appsv1beta1.StatefulSet:
		t.Spec.UpdateStrategy.Type = appsv1beta1.RollingUpdateStatefulSetStrategyType
	case *appsv1beta2.StatefulSet:
		t.Spec.UpdateStrategy.Type = appsv1beta2.RollingUpdateStatefulSetStrategyType
	case *appsv1.StatefulSet:
		t.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	}
}
