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
	"context"
	"fmt"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/stash/pkg/util"

	"gomodules.xyz/pointer"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	kutil "kmodules.xyz/client-go"
	apps_util "kmodules.xyz/client-go/apps/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	meta_util "kmodules.xyz/client-go/meta"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	ocapps_util "kmodules.xyz/openshift/client/clientset/versioned/typed/apps/v1/util"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

// applyStashLogic takes a workload and perform some processing on it if any backup or restore is configured for this workload.
func (c *StashController) applyStashLogic(w *wapi.Workload, caller string) (bool, error) {
	// check if restore is configured for this workload and perform respective operations
	modifiedByRestoreLogic, err := c.applyRestoreLogic(w, caller)
	if err != nil {
		return false, err
	}

	// check if backup is configured for this workload and perform respective operations
	modifiedByBackupLogic, err := c.applyBackupLogic(w, caller)
	if err != nil {
		return false, err
	}

	// apply changes of workload to original object
	err = wcs.ApplyWorkload(w.Object, w)
	if err != nil {
		return false, err
	}
	return modifiedByBackupLogic || modifiedByRestoreLogic, nil
}

func setRollingUpdate(w *wapi.Workload) error {
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
	default:
		return fmt.Errorf("unable to set RolingUpdateStrategy to workload. Reason: %s %s/%s of %s APIVersion is not supported", w.Kind, w.Namespace, w.Name, w.APIVersion)
	}
	return nil
}

func (c *StashController) ensureUnnecessaryConfigMapLockDeleted(w *wapi.Workload) error {
	// if the workload does not have any stash sidecar/init-container then
	// delete the respective ConfigMapLock if exist
	r := api_v1beta1.TargetRef{
		APIVersion: w.APIVersion,
		Kind:       w.Kind,
		Name:       w.Name,
		Namespace:  w.Namespace,
	}

	if !util.HasStashSidecar(w.Spec.Template.Spec.Containers) {
		// delete backup ConfigMap lock
		err := util.DeleteBackupConfigMapLock(c.kubeClient, r)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
		// backward compatibility
		err = util.DeleteConfigmapLock(c.kubeClient, w.Namespace, api_v1alpha1.LocalTypedReference{Kind: w.Kind, Name: w.Name, APIVersion: w.APIVersion})
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	}

	if !util.HasStashInitContainer(w.Spec.Template.Spec.InitContainers) {
		// delete restore ConfigMap lock
		err := util.DeleteRestoreConfigMapLock(c.kubeClient, r)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *StashController) getTotalHosts(target interface{}, driver api_v1beta1.Snapshotter) (*int32, error) {
	// for cluster backup/restore, target is nil. in this case, there is only one host
	var targetRef api_v1beta1.TargetRef
	var rep *int32
	if target == nil {
		return pointer.Int32P(1), nil
	}

	// target interface can be BackupTarget or RestoreTarget. We need to extract TargetRef from it.
	switch t := target.(type) {
	case *api_v1beta1.BackupTarget:
		if t.Replicas != nil {
			rep = t.Replicas
		}
		if t == nil {
			return pointer.Int32P(1), nil
		}
		targetRef = t.Ref

	case *api_v1beta1.RestoreTarget:
		if t == nil {
			return pointer.Int32P(1), nil
		}
		targetRef = t.Ref

		// for VolumeSnapshot, we consider each PVC as a separate host.
		// hence, number of host = replica * number of PVC in each replica
		if driver == api_v1beta1.VolumeSnapshotter {
			replica := int32(1)
			if t.Replicas != nil {
				replica = pointer.Int32(t.Replicas)
			}
			return pointer.Int32P(replica * int32(len(t.VolumeClaimTemplates))), nil
		}

		// if volumeClaimTemplates is specified when using Restic driver, restore is done through job.
		// stash creates restore job for each replica. hence, number of total host is the number of replicas.
		if len(t.VolumeClaimTemplates) != 0 || t.Replicas != nil {
			if t.Replicas == nil {
				return pointer.Int32P(1), nil
			} else {
				return t.Replicas, nil
			}
		}
	}

	if driver == api_v1beta1.VolumeSnapshotter {
		return c.getTotalHostForVolumeSnapshotter(targetRef, rep)
	} else {
		return c.getTotalHostForRestic(targetRef)
	}
}

func (c *StashController) getTotalHostForVolumeSnapshotter(targetRef api_v1beta1.TargetRef, replica *int32) (*int32, error) {
	switch targetRef.Kind {
	case apis.KindStatefulSet:
		ss, err := c.kubeClient.AppsV1().StatefulSets(targetRef.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if replica != nil {
			return pointer.Int32P(*replica * int32(len(ss.Spec.VolumeClaimTemplates))), err
		}
		return pointer.Int32P(pointer.Int32(ss.Spec.Replicas) * int32(len(ss.Spec.VolumeClaimTemplates))), err
	case apis.KindDeployment:
		deployment, err := c.kubeClient.AppsV1().Deployments(targetRef.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return countPVC(deployment.Spec.Template.Spec.Volumes), err

	case apis.KindDaemonSet:
		daemon, err := c.kubeClient.AppsV1().DaemonSets(targetRef.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return countPVC(daemon.Spec.Template.Spec.Volumes), err

	case apis.KindReplicaSet:
		rs, err := c.kubeClient.AppsV1().StatefulSets(targetRef.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return countPVC(rs.Spec.Template.Spec.Volumes), err

	case apis.KindReplicationController:
		rc, err := c.kubeClient.CoreV1().ReplicationControllers(targetRef.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return countPVC(rc.Spec.Template.Spec.Volumes), err

	default:
		return pointer.Int32P(1), nil
	}
}

func (c *StashController) getTotalHostForRestic(targetRef api_v1beta1.TargetRef) (*int32, error) {
	switch targetRef.Kind {
	// all replicas of StatefulSet will take backup/restore. so total number of hosts will be number of replicas.
	case apis.KindStatefulSet:
		ss, err := c.kubeClient.AppsV1().StatefulSets(targetRef.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return ss.Spec.Replicas, nil
	// all Daemon pod will take backup/restore. so total number of hosts will be number of ready replicas
	case apis.KindDaemonSet:
		dmn, err := c.kubeClient.AppsV1().DaemonSets(targetRef.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &dmn.Status.DesiredNumberScheduled, nil
	// for all other workloads, only one replica will take backup/restore. so number of total host will be 1
	default:
		return pointer.Int32P(1), nil
	}
}

func countPVC(volList []core.Volume) *int32 {
	var count int32
	for _, vol := range volList {
		if vol.PersistentVolumeClaim != nil {
			count++
		}
	}
	return &count
}

func (c *StashController) ensureImagePullSecrets(invokerMeta metav1.ObjectMeta, owner *metav1.OwnerReference) ([]core.LocalObjectReference, error) {
	operatorNamespace := meta.PodNamespace()
	if operatorNamespace == "" {
		operatorNamespace = "kube-system"
	}

	var imagePullSecrets []core.LocalObjectReference
	for i := range c.ImagePullSecrets {
		// get the respective secret from the operator namespace
		secret, err := c.kubeClient.CoreV1().Secrets(operatorNamespace).Get(context.TODO(), c.ImagePullSecrets[i], metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		// generate new image pull secret from the above secret
		newPullSecret := metav1.ObjectMeta{
			Name:      meta_util.ValidNameWithPrefixNSuffix(secret.Name, strings.ToLower(owner.Kind), invokerMeta.Name),
			Namespace: invokerMeta.Namespace,
		}
		// create the image pull secret if not present already
		_, _, err = core_util.CreateOrPatchSecret(context.TODO(), c.kubeClient, newPullSecret, func(in *core.Secret) *core.Secret {
			// set the invoker as the owner of this secret
			core_util.EnsureOwnerReference(&in.ObjectMeta, owner)
			in.Type = secret.Type
			in.Data = secret.Data
			return in
		}, metav1.PatchOptions{})

		if err != nil {
			return nil, err
		}
		imagePullSecrets = append(imagePullSecrets, core.LocalObjectReference{
			Name: newPullSecret.Name,
		})
	}
	return imagePullSecrets, nil
}

func (c *StashController) ensureLatestSidecarConfiguration(targetInfo invoker.BackupTargetInfo) error {
	obj, err := c.getTargetWorkload(targetInfo)
	if err != nil {
		return err
	}
	w, err := wcs.ConvertToWorkload(obj.DeepCopyObject())
	if err != nil {
		return err
	}
	targetRef := api_v1beta1.TargetRef{
		APIVersion: w.APIVersion,
		Kind:       w.Kind,
		Name:       w.Name,
		Namespace:  w.Namespace,
	}
	latestInvoker, err := util.FindLatestBackupInvoker(c.bcLister, targetRef)
	if err != nil {
		return err
	}
	inv, err := invoker.NewBackupInvoker(
		c.stashClient,
		latestInvoker.GetKind(),
		latestInvoker.GetName(),
		latestInvoker.GetNamespace(),
	)
	if err != nil {
		return err
	}
	err = c.ensureBackupSidecar(w, inv, targetInfo, apis.CallerController)
	if err != nil {
		return err
	}
	// apply the changes into the original object
	err = wcs.ApplyWorkload(w.Object, w)
	if err != nil {
		return err
	}
	return c.patchTargetWorkload(w, obj, targetInfo)
}

func (c *StashController) getTargetWorkload(targetInfo invoker.BackupTargetInfo) (runtime.Object, error) {
	switch targetInfo.Target.Ref.Kind {
	case apis.KindDeployment:
		dp, err := c.dpLister.Deployments(targetInfo.Target.Ref.Namespace).Get(targetInfo.Target.Ref.Name)
		if err != nil {
			return nil, err
		}
		dp.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind(apis.KindDeployment))
		return dp, nil
	case apis.KindDaemonSet:
		ds, err := c.dsLister.DaemonSets(targetInfo.Target.Ref.Namespace).Get(targetInfo.Target.Ref.Name)
		if err != nil {
			return nil, err
		}
		ds.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind(apis.KindDaemonSet))
		return ds, nil
	case apis.KindStatefulSet:
		ss, err := c.ssLister.StatefulSets(targetInfo.Target.Ref.Namespace).Get(targetInfo.Target.Ref.Name)
		if err != nil {
			return nil, err
		}
		ss.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind(apis.KindStatefulSet))
		return ss, nil
	case apis.KindReplicaSet:
		rs, err := c.rsLister.ReplicaSets(targetInfo.Target.Ref.Namespace).Get(targetInfo.Target.Ref.Name)
		if err != nil {
			return nil, err
		}
		rs.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind(apis.KindReplicaSet))
		return rs, nil
	case apis.KindReplicationController:
		rc, err := c.rcLister.ReplicationControllers(targetInfo.Target.Ref.Namespace).Get(targetInfo.Target.Ref.Name)
		if err != nil {
			return nil, err
		}
		rc.GetObjectKind().SetGroupVersionKind(core.SchemeGroupVersion.WithKind(apis.KindReplicationController))
		return rc, nil
	case apis.KindDeploymentConfig:
		dc, err := c.dcLister.DeploymentConfigs(targetInfo.Target.Ref.Namespace).Get(targetInfo.Target.Ref.Name)
		if err != nil {
			return nil, err
		}
		dc.GetObjectKind().SetGroupVersionKind(ocapps.GroupVersion.WithKind(apis.KindDeploymentConfig))
		return dc, nil
	default:
		return nil, fmt.Errorf("failed to get target workload. Reason: unknown kind %s", targetInfo.Target.Ref.Kind)
	}
}

func (c *StashController) patchTargetWorkload(w *wapi.Workload, oldObj runtime.Object, targetInfo invoker.BackupTargetInfo) error {
	switch targetInfo.Target.Ref.Kind {
	case apis.KindDeployment:
		_, verb, err := apps_util.PatchDeploymentObject(context.TODO(), c.kubeClient, oldObj.(*appsv1.Deployment), w.Object.(*appsv1.Deployment), metav1.PatchOptions{})
		if err != nil {
			return err
		}
		if verb == kutil.VerbPatched {
			return util.WaitUntilDeploymentReady(c.kubeClient, oldObj.(*appsv1.Deployment).ObjectMeta)
		}

	case apis.KindDaemonSet:
		_, verb, err := apps_util.PatchDaemonSetObject(context.TODO(), c.kubeClient, oldObj.(*appsv1.DaemonSet), w.Object.(*appsv1.DaemonSet), metav1.PatchOptions{})
		if err != nil {
			return err
		}
		if verb == kutil.VerbPatched {
			return util.WaitUntilDaemonSetReady(c.kubeClient, oldObj.(*appsv1.DaemonSet).ObjectMeta)
		}
	case apis.KindStatefulSet:
		_, verb, err := apps_util.PatchStatefulSetObject(context.TODO(), c.kubeClient, oldObj.(*appsv1.StatefulSet), w.Object.(*appsv1.StatefulSet), metav1.PatchOptions{})
		if err != nil {
			return err
		}
		if verb == kutil.VerbPatched {
			return util.WaitUntilStatefulSetReady(c.kubeClient, oldObj.(*appsv1.StatefulSet).ObjectMeta)
		}
	case apis.KindReplicaSet:
		_, verb, err := apps_util.PatchReplicaSetObject(context.TODO(), c.kubeClient, oldObj.(*appsv1.ReplicaSet), w.Object.(*appsv1.ReplicaSet), metav1.PatchOptions{})
		if err != nil {
			return err
		}
		if verb == kutil.VerbPatched {
			stageChanged, err := c.ensureWorkloadLatestState(w)
			if err != nil {
				return err
			}
			if stageChanged {
				return util.WaitUntilReplicaSetReady(c.kubeClient, oldObj.(*appsv1.ReplicaSet).ObjectMeta)
			}
		}
	case apis.KindReplicationController:
		_, verb, err := core_util.PatchRCObject(context.TODO(), c.kubeClient, oldObj.(*core.ReplicationController), w.Object.(*core.ReplicationController), metav1.PatchOptions{})
		if err != nil {
			return err
		}
		if verb == kutil.VerbPatched {
			stageChanged, err := c.ensureWorkloadLatestState(w)
			if err != nil {
				return err
			}
			if stageChanged {
				return util.WaitUntilReplicaSetReady(c.kubeClient, oldObj.(*appsv1.ReplicaSet).ObjectMeta)
			}
		}
	case apis.KindDeploymentConfig:
		_, verb, err := ocapps_util.PatchDeploymentConfigObject(context.TODO(), c.ocClient, oldObj.(*ocapps.DeploymentConfig), w.Object.(*ocapps.DeploymentConfig), metav1.PatchOptions{})
		if err != nil {
			return err
		}
		if verb == kutil.VerbPatched {
			return util.WaitUntilDeploymentConfigReady(c.ocClient, oldObj.(*ocapps.DeploymentConfig).ObjectMeta)
		}
	default:
		return fmt.Errorf("unkown workload kind: %s", targetInfo.Target.Ref.Kind)
	}
	return nil
}
