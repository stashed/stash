package controller

import (
	"fmt"
	"strings"

	"github.com/appscode/stash/apis"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/resolve"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
)

const (
	StashDefaultVolume = "stash-volume"
)

func (c *StashController) initPVCWatcher() {
	c.pvcInformer = c.kubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer()
	c.pvcQueue = queue.New(apis.KindPersistentVolumeClaim, c.MaxNumRequeues, c.NumThreads, c.processPVCKey)
	c.pvcInformer.AddEventHandler(queue.DefaultEventHandler(c.pvcQueue.GetQueue()))
	c.pvcLister = c.kubeInformerFactory.Core().V1().PersistentVolumeClaims().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) processPVCKey(key string) error {
	obj, exists, err := c.pvcInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a PersistentVolumeClaim, so that we will see a delete for one pvc.
		glog.Warningf("PersistentVolumeClaim %s does not exist anymore\n", key)
	} else {
		glog.Infof("Sync/Add/Update for PersistentVolumeClaim %s", key)

		pvc := obj.(*core.PersistentVolumeClaim).DeepCopy()
		pvc.GetObjectKind().SetGroupVersionKind(core.SchemeGroupVersion.WithKind(apis.KindPersistentVolumeClaim))

		err := c.applyBackupAnnotationLogicForPVC(pvc)
		if err != nil {
			return err
		}

	}
	return nil
}

func (c *StashController) applyBackupAnnotationLogicForPVC(pvc *core.PersistentVolumeClaim) error {
	targetRef, err := reference.GetReference(scheme.Scheme, pvc)
	if err != nil {
		return fmt.Errorf("failed to create reference of %s %s/%s. Reason: %v", pvc.Kind, pvc.Namespace, pvc.Name, err)
	}

	// if pvc has backup annotations then ensure respective Repository and BackupConfiguration
	if meta_util.HasKey(pvc.Annotations, api_v1beta1.KeyBackupConfigurationTemplate) &&
		meta_util.HasKey(pvc.Annotations, api_v1beta1.KeyMountPath) &&
		meta_util.HasKey(pvc.Annotations, api_v1beta1.KeyTargetDirectories) {
		// backup annotations found. so, we have to ensure Repository and BackupConfiguration from BackupConfigurationTemplate
		backupTemplateName, err := meta_util.GetStringValue(pvc.Annotations, api_v1beta1.KeyBackupConfigurationTemplate)
		if err != nil {
			return err
		}
		mountPath, err := meta_util.GetStringValue(pvc.Annotations, api_v1beta1.KeyMountPath)
		if err != nil {
			return err
		}
		directories, err := meta_util.GetStringValue(pvc.Annotations, api_v1beta1.KeyTargetDirectories)
		if err != nil {
			return err
		}

		backupTemplate, err := c.stashClient.StashV1beta1().BackupConfigurationTemplates().Get(backupTemplateName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// resolve BackupConfigurationTemplate's variables
		inputs := make(map[string]string, 0)
		inputs[apis.TargetAPIVersion] = pvc.APIVersion
		inputs[apis.TargetKind] = strings.ToLower(pvc.Kind)
		inputs[apis.TargetName] = pvc.Name
		inputs[apis.TargetNamespace] = pvc.Namespace

		err = resolve.ResolveBackend(&backupTemplate.Spec.Backend, inputs)
		if err != nil {
			return err
		}

		// ensure Repository crd
		err = c.ensureRepository(backupTemplate, targetRef)
		if err != nil {
			return err
		}

		// ensure BackupConfiguration crd
		volumeMounts := []core.VolumeMount{
			{
				Name:      StashDefaultVolume,
				MountPath: mountPath,
			},
		}
		err = c.ensureBackupConfiguration(backupTemplate, strings.Split(directories, ","), volumeMounts, targetRef)
		if err != nil {
			return err
		}

	} else {
		// pvc does not have backup annotations. it might be removed or was never added.
		// if respective BackupConfiguration exist then backup annotations has been removed.
		// in this case, we have to remove the BackupConfiguration too.
		// however, we will keep Repository crd as it is required for restore.
		_, err := c.stashClient.StashV1beta1().BackupConfigurations(pvc.Namespace).Get(getBackupConfigurationName(targetRef), metav1.GetOptions{})
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
		// BackupConfiguration exist. so, we have to remove it.
		err = c.stashClient.StashV1beta1().BackupConfigurations(pvc.Namespace).Delete(getBackupConfigurationName(targetRef), meta_util.DeleteInBackground())
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}
