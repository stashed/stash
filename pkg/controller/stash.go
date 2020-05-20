/*
Copyright The Stash Authors.

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

package controller

import (
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/types"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

// applyStashLogic takes an workload and perform some processing on it if any backup or restore is configured for this workload.
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

// applyRestoreLogic check if  RestoreSession is configured for this workload
// and perform operation accordingly
func (c *StashController) applyRestoreLogic(w *wapi.Workload, caller string) (bool, error) {
	// detect old RestoreSession from annotations if it does exist.
	oldRestore, err := util.GetAppliedRestoreSession(w.Annotations)
	if err != nil {
		return false, err
	}
	// find existing pending RestoreSession for this workload
	newRestore, err := util.FindRestoreSession(c.restoreSessionLister, w)
	if err != nil {
		return false, err
	}
	// if RestoreSession currently exist for this workload but it is not same as old one,
	// this means RestoreSession has been newly created/updated.
	// in this case, we have to add/update the init-container accordingly.
	if newRestore != nil && !util.RestoreSessionEqual(oldRestore, newRestore) {
		err := c.ensureRestoreInitContainer(w, newRestore, caller)
		// write init-container injection failure/success event
		ref, rerr := util.GetWorkloadReference(w)
		if err != nil && rerr != nil {
			return false, err
		} else if err != nil && rerr == nil {
			return false, c.handleInitContainerInjectionFailure(ref, newRestore, err)
		} else if err == nil && rerr != nil {
			return true, nil
		}
		return true, c.handleInitContainerInjectionSuccess(ref, newRestore)
	} else if oldRestore != nil && newRestore == nil {
		// there was RestoreSession before but currently it does not exist.
		// this means RestoreSession has been removed.
		// in this case, we have to delete the restore init-container
		// and remove respective annotations from the workload  and respective ConfigMapLock.
		c.ensureRestoreInitContainerDeleted(w)
		// write init-container deletion failure/success event
		ref, rerr := util.GetWorkloadReference(w)
		if rerr != nil {
			return true, nil
		}
		return true, c.handleInitContainerDeletionSuccess(ref)
	}

	return false, nil
}

// applyBackupLogic check if Backup annotations or BackupConfiguration is configured for this workload
// and perform operation accordingly
func (c *StashController) applyBackupLogic(w *wapi.Workload, caller string) (bool, error) {
	//Don't create repository, BackupConfiguration stuff when the caller is webhook to make the webhooks side effect free.
	if caller != apis.CallerWebhook {
		// check if the workload has backup annotations and perform respective operation accordingly
		err := c.applyBackupAnnotationLogic(w)
		if err != nil {
			return false, err
		}
	}
	// check if any BackupBatch exits for this workload. if exist then inject sidecar container
	modified, err := c.applyBackupBatchLogic(w, caller)
	if err != nil {
		return false, err
	}

	// if no BackupBatch is configured then check any BackupConfiguration exist for this workload.
	// if exist then inject sidecar container
	if !modified {
		modified, err = c.applyBackupConfigurationLogic(w, caller)
		if err != nil {
			return false, err
		}
	}

	// if no BackupConfiguration is configured then check if Restic is configured (backward compatibility)
	if !modified {
		return c.applyResticLogic(w, caller)
	}
	return modified, nil
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
	}

	if !util.HasStashSidecar(w.Spec.Template.Spec.Containers) {
		// delete backup ConfigMap lock
		err := util.DeleteBackupConfigMapLock(c.kubeClient, w.Namespace, r)
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
		err := util.DeleteRestoreConfigMapLock(c.kubeClient, w.Namespace, r)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *StashController) getTotalHosts(target interface{}, namespace string, driver api_v1beta1.Snapshotter) (*int32, error) {

	// for cluster backup/restore, target is nil. in this case, there is only one host
	var targetRef api_v1beta1.TargetRef
	var rep *int32
	if target == nil {
		return types.Int32P(1), nil
	}

	// target interface can be BackupTarget or RestoreTarget. We need to extract TargetRef from it.
	switch t := target.(type) {
	case *api_v1beta1.BackupTarget:
		if t.Replicas != nil {
			rep = t.Replicas
		}
		if t == nil {
			return types.Int32P(1), nil
		}
		targetRef = t.Ref

	case *api_v1beta1.RestoreTarget:
		if t == nil {
			return types.Int32P(1), nil
		}
		targetRef = t.Ref

		// for VolumeSnapshot, we consider each PVC as a separate host.
		// hence, number of host = replica * number of PVC in each replica
		if driver == api_v1beta1.VolumeSnapshotter {
			replica := int32(1)
			if t.Replicas != nil {
				replica = types.Int32(t.Replicas)
			}
			return types.Int32P(replica * int32(len(t.VolumeClaimTemplates))), nil
		}

		// if volumeClaimTemplates is specified when using Restic driver, restore is done through job.
		// stash creates restore job for each replica. hence, number of total host is the number of replicas.
		if len(t.VolumeClaimTemplates) != 0 || t.Replicas != nil {
			if t.Replicas == nil {
				return types.Int32P(1), nil
			} else {
				return t.Replicas, nil
			}
		}
	}

	if driver == api_v1beta1.VolumeSnapshotter {
		return c.getTotalHostForVolumeSnapshotter(targetRef, namespace, rep)
	} else {
		return c.getTotalHostForRestic(targetRef, namespace)
	}
}

func (c *StashController) getTotalHostForVolumeSnapshotter(targetRef api_v1beta1.TargetRef, namespace string, replica *int32) (*int32, error) {
	switch targetRef.Kind {
	case apis.KindStatefulSet:
		ss, err := c.kubeClient.AppsV1().StatefulSets(namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if replica != nil {
			return types.Int32P(*replica * int32(len(ss.Spec.VolumeClaimTemplates))), err
		}
		return types.Int32P(types.Int32(ss.Spec.Replicas) * int32(len(ss.Spec.VolumeClaimTemplates))), err
	case apis.KindDeployment:
		deployment, err := c.kubeClient.AppsV1().Deployments(namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return countPVC(deployment.Spec.Template.Spec.Volumes), err

	case apis.KindDaemonSet:
		daemon, err := c.kubeClient.AppsV1().DaemonSets(namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return countPVC(daemon.Spec.Template.Spec.Volumes), err

	case apis.KindReplicaSet:
		rs, err := c.kubeClient.AppsV1().StatefulSets(namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return countPVC(rs.Spec.Template.Spec.Volumes), err

	case apis.KindReplicationController:
		rc, err := c.kubeClient.CoreV1().ReplicationControllers(namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return countPVC(rc.Spec.Template.Spec.Volumes), err

	default:
		return types.Int32P(1), nil
	}
}

func (c *StashController) getTotalHostForRestic(targetRef api_v1beta1.TargetRef, namespace string) (*int32, error) {
	switch targetRef.Kind {
	// all replicas of StatefulSet will take backup/restore. so total number of hosts will be number of replicas.
	case apis.KindStatefulSet:
		ss, err := c.kubeClient.AppsV1().StatefulSets(namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return ss.Spec.Replicas, nil
	// all Daemon pod will take backup/restore. so total number of hosts will be number of ready replicas
	case apis.KindDaemonSet:
		dmn, err := c.kubeClient.AppsV1().DaemonSets(namespace).Get(targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &dmn.Status.DesiredNumberScheduled, nil
	// for all other workloads, only one replica will take backup/restore. so number of total host will be 1
	default:
		return types.Int32P(1), nil
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
