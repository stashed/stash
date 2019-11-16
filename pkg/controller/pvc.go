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
	"strings"

	"stash.appscode.dev/stash/apis"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/resolve"

	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	kutil "kmodules.xyz/client-go"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
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
	if meta_util.HasKey(pvc.Annotations, api_v1beta1.KeyBackupBlueprint) {
		// backup annotations found. so, we have to ensure Repository and BackupConfiguration from BackupBlueprint
		verb, err := c.ensureAutoBackupResourcesForPVC(pvc, targetRef)
		if err != nil {
			return c.handleAutoBackupResourcesCreationFailure(targetRef, err)
		}
		if verb != kutil.VerbUnchanged {
			return c.handleAutoBackupResourcesCreationSuccess(targetRef)
		}
	} else {
		// pvc does not have backup annotations. it might be removed or was never added.
		// if respective BackupConfiguration exist then backup annotations has been removed.
		// in this case, we have to remove the BackupConfiguration too.
		// however, we will keep Repository crd as it is required for restore.
		verb, err := c.ensureAutoBackupResourcesDeleted(targetRef, pvc.Namespace, targetRef.Kind)
		if err != nil {
			return c.handleAutoBackupResourcesDeletionFailure(targetRef, err)
		}
		if verb != kutil.VerbUnchanged {
			return c.handleAutoBackupResourcesDeletionSuccess(targetRef)
		}
	}
	return nil
}

func (c *StashController) ensureAutoBackupResourcesForPVC(pvc *core.PersistentVolumeClaim, targetRef *core.ObjectReference) (kutil.VerbType, error) {

	backupBlueprintName, err := meta_util.GetStringValue(pvc.Annotations, api_v1beta1.KeyBackupBlueprint)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	backupBlueprint, err := c.stashClient.StashV1beta1().BackupBlueprints().Get(backupBlueprintName, metav1.GetOptions{})
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	// resolve BackupBlueprint's variables
	inputs := make(map[string]string)
	inputs[apis.TargetAPIVersion] = pvc.APIVersion
	inputs[apis.TargetKind] = strings.ToLower(pvc.Kind)
	inputs[apis.TargetName] = pvc.Name
	inputs[apis.TargetNamespace] = pvc.Namespace

	err = resolve.ResolveBackupBlueprint(backupBlueprint, inputs)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	// ensure Repository crd
	verb1, err := c.ensureRepository(backupBlueprint, targetRef, targetRef.Kind)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	// ensure BackupConfiguration crd. For stand-alone PVC backup, we don't need to specify target paths and volumeMounts.
	// Stash will use default target path and mount path.
	verb2, err := c.ensureBackupConfiguration(backupBlueprint, nil, nil, targetRef, targetRef.Kind)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	// if both of the verb is unchanged then no create/update happened to the auto backup resources
	if verb1 == kutil.VerbUnchanged || verb2 == kutil.VerbUnchanged {
		return kutil.VerbUnchanged, nil
	}
	// auto backup resources has been created/updated
	// we will use this information to write event to PVC
	// so, "created" or "updated" verb has same effect to the end result
	// we can return any of them.
	return kutil.VerbCreated, nil
}
