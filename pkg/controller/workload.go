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
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/util"

	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
	meta_util "kmodules.xyz/client-go/meta"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

type workloadReconciler struct {
	ctrl     *StashController
	logger   klog.Logger
	workload *wapi.Workload
}
type invokerOptions struct {
	ctrl       *StashController
	workload   *wapi.Workload
	caller     string
	logger     klog.Logger
	oldInvoker unstructured.Unstructured
	newInvoker unstructured.Unstructured
	targetRef  api_v1beta1.TargetRef
}

// reconcileWorkload takes a workload and perform some processing on it if any backup or restore is configured for this workload.
func (r *workloadReconciler) reconcile(caller string) error {
	var err error

	if r.workload.DeletionTimestamp != nil {
		r.logger.V(4).Info("Skipping processing event",
			apis.KeyReason, "Resource is being deleted",
		)
		return nil
	}

	// =============== restore ==============
	opt, err := r.newRestoreOptions(caller)
	if err != nil {
		return err
	}
	yes, err := opt.restoreConfigured()
	if err != nil {
		return err
	}
	if yes {
		if err := opt.injectRestoreInitContainer(); err != nil {
			return err
		}
	} else if opt.restoreWasConfiguredBefore() {
		if err := opt.removeRestoreInitContainer(); err != nil {
			return err
		}
	}

	// ================= backup  ================
	opt, err = r.newBackupOptions(caller)
	if err != nil {
		return err
	}
	yes, err = opt.backupConfigured()
	if err != nil {
		return err
	}
	if yes {
		if err := opt.injectBackupSidecar(); err != nil {
			return err
		}
	} else if opt.backupWasConfiguredBefore() {
		if err := opt.removeBackupSidecar(); err != nil {
			return err
		}
	}
	return nil
}

func (r *workloadReconciler) newBackupOptions(caller string) (*invokerOptions, error) {
	targetRef := api_v1beta1.TargetRef{
		APIVersion: r.workload.APIVersion,
		Kind:       r.workload.Kind,
		Name:       r.workload.Name,
		Namespace:  r.workload.Namespace,
	}

	oldInvoker, err := util.ExtractAppliedBackupInvokerFromAnnotation(r.workload.Annotations)
	if err != nil {
		return nil, err
	}

	newInvoker, err := util.FindLatestBackupInvoker(r.ctrl.bcLister, targetRef)
	if err != nil {
		return nil, err
	}
	return &invokerOptions{
		ctrl:       r.ctrl,
		workload:   r.workload,
		caller:     caller,
		logger:     r.logger,
		oldInvoker: oldInvoker,
		newInvoker: newInvoker,
		targetRef:  targetRef,
	}, nil
}

func (opt *invokerOptions) backupConfigured() (bool, error) {
	if opt.newInvoker.Object == nil {
		return false, nil
	}
	equal, err := util.InvokerEqual(opt.oldInvoker, opt.newInvoker)
	if err != nil {
		return false, err
	}
	return !equal, nil
}

func (opt *invokerOptions) backupWasConfiguredBefore() bool {
	return opt.oldInvoker.Object != nil && opt.newInvoker.Object == nil
}

func (opt *invokerOptions) injectBackupSidecar() error {
	inv, err := invoker.NewBackupInvoker(
		opt.ctrl.stashClient,
		opt.newInvoker.GetKind(),
		opt.newInvoker.GetName(),
		opt.newInvoker.GetNamespace(),
	)
	if err != nil {
		return err
	}

	for i, targetInfo := range inv.GetTargetInfo() {
		if util.IsBackupTarget(targetInfo.Target, opt.targetRef, inv.GetObjectMeta().Namespace) {
			e, err := opt.ctrl.newSidecarExecutor(inv, opt.workload, i, opt.caller)
			if err != nil {
				return opt.handleSidecarInjectionFailure(inv, err)
			}
			obj, verb, err := e.Ensure()
			if err != nil {
				return opt.handleSidecarInjectionFailure(inv, err)
			}
			if verb != kutil.VerbUnchanged {
				opt.workload, err = wcs.ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				return opt.handleSidecarInjectionSuccess(inv)
			}
			return nil
		}
	}
	return nil
}

func (opt *invokerOptions) removeBackupSidecar() error {
	inv, err := invoker.NewBackupInvoker(
		opt.ctrl.stashClient,
		opt.oldInvoker.GetKind(),
		opt.oldInvoker.GetName(),
		opt.oldInvoker.GetNamespace(),
	)
	if err != nil {
		return err
	}

	for i, targetInfo := range inv.GetTargetInfo() {
		if util.IsBackupTarget(targetInfo.Target, opt.targetRef, inv.GetObjectMeta().Namespace) {
			e, err := opt.ctrl.newSidecarExecutor(inv, opt.workload, i, opt.caller)
			if err != nil {
				return opt.handleSidecarDeletionFailure(err)
			}
			obj, verb, err := e.Cleanup()
			if err != nil {
				if kerr.IsNotFound(err) {
					return nil
				}
				return opt.handleSidecarDeletionFailure(err)
			}
			if verb != kutil.VerbUnchanged {
				opt.workload, err = wcs.ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				return opt.handleSidecarDeletionSuccess()
			}
			return nil
		}
	}
	return nil
}

func (r *workloadReconciler) newRestoreOptions(caller string) (*invokerOptions, error) {
	targetRef := api_v1beta1.TargetRef{
		APIVersion: r.workload.APIVersion,
		Kind:       r.workload.Kind,
		Name:       r.workload.Name,
		Namespace:  r.workload.Namespace,
	}

	oldInvoker, err := util.ExtractAppliedRestoreInvokerFromAnnotation(r.workload.Annotations)
	if err != nil {
		return nil, err
	}

	newInvoker, err := util.FindLatestRestoreInvoker(r.ctrl.restoreSessionLister, targetRef)
	if err != nil {
		return nil, err
	}
	return &invokerOptions{
		ctrl:       r.ctrl,
		workload:   r.workload,
		caller:     caller,
		logger:     r.logger,
		oldInvoker: oldInvoker,
		newInvoker: newInvoker,
		targetRef:  targetRef,
	}, nil
}

func (opt *invokerOptions) restoreConfigured() (bool, error) {
	if opt.newInvoker.Object == nil {
		return false, nil
	}
	equal, err := util.InvokerEqual(opt.oldInvoker, opt.newInvoker)
	if err != nil {
		return false, err
	}
	return !equal, nil
}

func (opt *invokerOptions) restoreWasConfiguredBefore() bool {
	return opt.oldInvoker.Object != nil && opt.newInvoker.Object == nil
}

func (opt *invokerOptions) injectRestoreInitContainer() error {
	inv, err := invoker.NewRestoreInvoker(
		opt.ctrl.kubeClient,
		opt.ctrl.stashClient,
		opt.newInvoker.GetKind(),
		opt.newInvoker.GetName(),
		opt.newInvoker.GetNamespace(),
	)
	if err != nil {
		return err
	}

	for i, targetInfo := range inv.GetTargetInfo() {
		if util.IsRestoreTarget(targetInfo.Target, opt.targetRef, inv.GetObjectMeta().Namespace) {
			e, err := opt.ctrl.newInitContainerExecutor(inv, opt.workload, i, opt.caller)
			if err != nil {
				return opt.handleInitContainerInjectionFailure(inv, err)
			}
			obj, verb, err := e.Ensure()
			if err != nil {
				return opt.handleInitContainerInjectionFailure(inv, err)
			}
			if verb != kutil.VerbUnchanged {
				opt.workload, err = wcs.ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				return opt.handleInitContainerInjectionSuccess(inv)
			}
			return nil
		}
	}
	return nil
}

func (opt *invokerOptions) removeRestoreInitContainer() error {
	inv, err := invoker.NewRestoreInvoker(
		opt.ctrl.kubeClient,
		opt.ctrl.stashClient,
		opt.oldInvoker.GetKind(),
		opt.oldInvoker.GetName(),
		opt.oldInvoker.GetNamespace(),
	)
	if err != nil {
		return err
	}

	for i, targetInfo := range inv.GetTargetInfo() {
		if util.IsRestoreTarget(targetInfo.Target, opt.targetRef, inv.GetObjectMeta().Namespace) {
			e, err := opt.ctrl.newInitContainerExecutor(inv, opt.workload, i, opt.caller)
			if err != nil {
				return opt.handleInitContainerDeletionFailure(err)
			}
			obj, verb, err := e.Cleanup()
			if err != nil {
				if kerr.IsNotFound(err) {
					return nil
				}
				return opt.handleInitContainerDeletionFailure(err)
			}
			if verb != kutil.VerbUnchanged {
				opt.workload, err = wcs.ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				return opt.handleInitContainerDeletionSuccess()
			}
			return nil
		}
	}
	return nil
}

func (c *StashController) sendEventToWorkloadQueue(kind, namespace, resourceName string) error {
	switch kind {
	case wapi.KindDeployment:
		if resource, err := c.dpLister.Deployments(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.dpQueue.GetQueue().Add(key)
			}
			return err
		}
	case wapi.KindDaemonSet:
		if resource, err := c.dsLister.DaemonSets(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.dsQueue.GetQueue().Add(key)
			}
			return err
		}
	case wapi.KindStatefulSet:
		if resource, err := c.ssLister.StatefulSets(namespace).Get(resourceName); err == nil {
			key, err := cache.MetaNamespaceKeyFunc(resource)
			if err == nil {
				c.ssQueue.GetQueue().Add(key)
			}
		}
	case wapi.KindDeploymentConfig:
		if c.ocClient != nil && c.dcLister != nil {
			if resource, err := c.dcLister.DeploymentConfigs(namespace).Get(resourceName); err == nil {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dcQueue.GetQueue().Add(key)
				}
				return err
			}
		}
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

func (c *StashController) getTargetWorkload(targetRef api_v1beta1.TargetRef) (runtime.Object, error) {
	switch targetRef.Kind {
	case apis.KindDeployment:
		dp, err := c.dpLister.Deployments(targetRef.Namespace).Get(targetRef.Name)
		if err != nil {
			return nil, err
		}
		dp.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind(apis.KindDeployment))
		return dp, nil
	case apis.KindDaemonSet:
		ds, err := c.dsLister.DaemonSets(targetRef.Namespace).Get(targetRef.Name)
		if err != nil {
			return nil, err
		}
		ds.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind(apis.KindDaemonSet))
		return ds, nil
	case apis.KindStatefulSet:
		ss, err := c.ssLister.StatefulSets(targetRef.Namespace).Get(targetRef.Name)
		if err != nil {
			return nil, err
		}
		ss.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind(apis.KindStatefulSet))
		return ss, nil
	case apis.KindDeploymentConfig:
		dc, err := c.dcLister.DeploymentConfigs(targetRef.Namespace).Get(targetRef.Name)
		if err != nil {
			return nil, err
		}
		dc.GetObjectKind().SetGroupVersionKind(ocapps.GroupVersion.WithKind(apis.KindDeploymentConfig))
		return dc, nil
	default:
		return nil, fmt.Errorf("failed to get target workload. Reason: unknown kind %s", targetRef.Kind)
	}
}

func (opt *invokerOptions) handleSidecarInjectionFailure(inv invoker.BackupInvoker, err error) error {
	opt.logger.Error(err, "Failed to inject stash sidecar")

	// Failed to inject stash sidecar. So, set "StashSidecarInjected" condition to "False".
	cerr := conditions.SetSidecarInjectedConditionToFalse(inv, opt.targetRef, err)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		opt.ctrl.kubeClient,
		eventer.EventSourceWorkloadController,
		opt.workload.Object,
		corev1.EventTypeWarning,
		eventer.EventReasonSidecarInjectionFailed,
		fmt.Sprintf("Failed to inject stash sidecar into %s %s/%s. Reason: %v",
			opt.targetRef.Kind,
			opt.targetRef.Namespace,
			opt.targetRef.Name,
			err,
		),
	)
	return errors.NewAggregate([]error{err2, cerr})
}

func (opt *invokerOptions) handleSidecarInjectionSuccess(inv invoker.BackupInvoker) error {
	opt.logger.Info("Successfully injected stash sidecar")

	// Set "StashSidecarInjected" condition to "True"
	cerr := conditions.SetSidecarInjectedConditionToTrue(inv, opt.targetRef)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		opt.ctrl.kubeClient,
		eventer.EventSourceWorkloadController,
		opt.workload.Object,
		corev1.EventTypeNormal,
		eventer.EventReasonSidecarInjectionSucceeded,
		fmt.Sprintf("Successfully injected stash sidecar into %s %s/%s.",
			opt.targetRef.Kind,
			opt.targetRef.Namespace,
			opt.targetRef.Name,
		),
	)
	return errors.NewAggregate([]error{err2, cerr})
}

func (opt *invokerOptions) handleSidecarDeletionSuccess() error {
	opt.logger.Info("Successfully removed stash sidecar")

	// write event to respective resource
	_, err := eventer.CreateEvent(
		opt.ctrl.kubeClient,
		eventer.EventSourceWorkloadController,
		opt.workload.Object,
		corev1.EventTypeNormal,
		eventer.EventReasonSidecarDeletionSucceeded,
		fmt.Sprintf("Successfully removed stash sidecar from %s %s/%s.",
			opt.targetRef.Kind,
			opt.targetRef.Namespace,
			opt.targetRef.Name,
		),
	)
	return err
}

func (opt *invokerOptions) handleSidecarDeletionFailure(err error) error {
	opt.logger.Error(err, "Failed to remove stash sidecar")

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		opt.ctrl.kubeClient,
		eventer.EventSourceWorkloadController,
		opt.workload.Object,
		corev1.EventTypeWarning,
		eventer.EventReasonSidecarDeletionFailed,
		fmt.Sprintf("Failed to remove stash sidecar from %s %s/%s. Reason: %v",
			opt.targetRef.Kind,
			opt.targetRef.Namespace,
			opt.targetRef.Name,
			err,
		),
	)
	return err2
}

func (opt *invokerOptions) handleInitContainerInjectionFailure(inv invoker.RestoreInvoker, err error) error {
	opt.logger.Error(err, "Failed to inject stash init-container")

	// Set "StashInitContainerInjected" condition to "False"
	cerr := conditions.SetInitContainerInjectedConditionToFalse(inv, &opt.targetRef, err)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		opt.ctrl.kubeClient,
		eventer.EventSourceWorkloadController,
		opt.workload.Object,
		core.EventTypeWarning,
		eventer.EventReasonInitContainerInjectionFailed,
		fmt.Sprintf("Failed to inject stash init-container into %s %s/%s. Reason: %v",
			opt.targetRef.Kind,
			opt.targetRef.Namespace,
			opt.targetRef.Name,
			err,
		),
	)
	return errors.NewAggregate([]error{err2, cerr})
}

func (opt *invokerOptions) handleInitContainerInjectionSuccess(inv invoker.RestoreInvoker) error {
	opt.logger.Info("Successfully injected stash init-container")

	// Set "StashInitContainerInjected" condition to "True"
	cerr := conditions.SetInitContainerInjectedConditionToTrue(inv, &opt.targetRef)

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		opt.ctrl.kubeClient,
		eventer.EventSourceWorkloadController,
		opt.workload.Object,
		core.EventTypeNormal,
		eventer.EventReasonInitContainerInjectionSucceeded,
		fmt.Sprintf("Successfully injected stash init-container into %s %s/%s.",
			opt.targetRef.Kind,
			opt.targetRef.Namespace,
			opt.targetRef.Name,
		),
	)
	return errors.NewAggregate([]error{err2, cerr})
}

func (opt *invokerOptions) handleInitContainerDeletionSuccess() error {
	opt.logger.Info("Successfully removed stash init-container")

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		opt.ctrl.kubeClient,
		eventer.EventSourceWorkloadController,
		opt.workload.Object,
		core.EventTypeNormal,
		eventer.EventReasonInitContainerDeletionSucceeded,
		fmt.Sprintf("Successfully removed stash init-container from %s %s/%s.",
			opt.targetRef.Kind,
			opt.targetRef.Namespace,
			opt.targetRef.Name,
		),
	)
	return err2
}

func (opt *invokerOptions) handleInitContainerDeletionFailure(err error) error {
	opt.logger.Error(err, "Failed to remove stash init-container")

	// write event to respective resource
	_, err2 := eventer.CreateEvent(
		opt.ctrl.kubeClient,
		eventer.EventSourceWorkloadController,
		opt.workload.Object,
		corev1.EventTypeWarning,
		eventer.EventReasonSidecarDeletionFailed,
		fmt.Sprintf("Failed to remove stash-init init-container from %s %s/%s. Reason: %v",
			opt.targetRef.Kind,
			opt.targetRef.Namespace,
			opt.targetRef.Name,
			err,
		),
	)
	return err2
}
