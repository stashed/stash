package controller

import (
	"fmt"

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
	"stash.appscode.dev/stash/apis"
	api_v1alpha1 "stash.appscode.dev/stash/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
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
		if err != nil {
			return false, err
		}
		return true, nil
	} else if oldRestore != nil && newRestore == nil {
		// there was RestoreSession before but currently it does not exist.
		// this means RestoreSession has been removed.
		// in this case, we have to delete the restore init-container
		// and remove respective annotations from the workload  and respective ConfigMapLock.
		err := c.ensureRestoreInitContainerDeleted(w, oldRestore)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// applyBackupLogic check if Backup annotations or BackupConfiguration is configured for this workload
// and perform operation accordingly
func (c *StashController) applyBackupLogic(w *wapi.Workload, caller string) (bool, error) {
	//Don't create repository, BackupConfiguration stuff when the caller is webhook to make the webhooks side effect free.
	if caller != util.CallerWebhook {
		// check if the workload has backup annotations and perform respective operation accordingly
		err := c.applyBackupAnnotationLogic(w)
		if err != nil {
			return false, err
		}
	}
	// check if any BackupConfiguration exist for this workload. if exist then inject sidecar container
	modified, err := c.applyBackupConfigurationLogic(w, caller)
	if err != nil {
		return false, err
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

	if !hasStashSidecar(w.Spec.Template.Spec.Containers) {
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

	if !hasStashInitContainer(w.Spec.Template.Spec.InitContainers) {
		// delete restore ConfigMap lock
		err := util.DeleteRestoreConfigMapLock(c.kubeClient, w.Namespace, r)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func hasStashContainer(w *wapi.Workload) bool {
	return hasStashSidecar(w.Spec.Template.Spec.Containers) || hasStashInitContainer(w.Spec.Template.Spec.InitContainers)
}

func hasStashSidecar(containers []core.Container) bool {
	// check if the workload has stash sidecar container
	for _, c := range containers {
		if c.Name == util.StashContainer {
			return true
		}
	}
	return false
}

func hasStashInitContainer(containers []core.Container) bool {
	// check if the workload has stash init-container
	for _, c := range containers {
		if c.Name == util.StashInitContainer {
			return true
		}
	}
	return false
}

func (c *StashController) getTotalHosts(target interface{}, namespace string) (*int32, error) {

	// for cluster backup/restore, target is nil. in this case, there is only one host
	if target == nil {
		return types.Int32P(1), nil
	}

	var targetRef api_v1beta1.TargetRef

	switch target.(type) {
	case *api_v1beta1.BackupTarget:
		targetRef = target.(*api_v1beta1.BackupTarget).Ref
	case *api_v1beta1.RestoreTarget:
		targetRef = target.(*api_v1beta1.RestoreTarget).Ref
	}

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
		return &dmn.Status.NumberReady, nil
	// for all other workloads, only one replica will take backup/restore. so number of total host will be 1
	default:
		return types.Int32P(1), nil
	}
}
