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

	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/reference"
	kutil "kmodules.xyz/client-go"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	catalog_scheme "kmodules.xyz/custom-resources/client/clientset/versioned/scheme"
)

func (c *StashController) initAppBindingWatcher() {
	c.abInformer = c.appCatalogInformerFactory.Appcatalog().V1alpha1().AppBindings().Informer()
	c.abQueue = queue.New(apis.KindAppBinding, c.MaxNumRequeues, c.NumThreads, c.processAppBindingKey)
	c.abInformer.AddEventHandler(queue.DefaultEventHandler(c.abQueue.GetQueue()))
	c.abLister = c.appCatalogInformerFactory.Appcatalog().V1alpha1().AppBindings().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) processAppBindingKey(key string) error {
	obj, exists, err := c.abInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a AppBinding, so that we will see a delete for one databases.
		glog.Warningf("AppBinding %s does not exist anymore\n", key)
	} else {
		glog.Infof("Sync/Add/Update for AppBinding %s", key)

		ab := obj.(*appCatalog.AppBinding).DeepCopy()
		ab.GetObjectKind().SetGroupVersionKind(core.SchemeGroupVersion.WithKind(apis.KindAppBinding))
		return c.applyBackupAnnotationLogicForAppBinding(ab)
	}
	return nil
}

func (c *StashController) applyBackupAnnotationLogicForAppBinding(ab *appCatalog.AppBinding) error {
	targetRef, err := reference.GetReference(catalog_scheme.Scheme, ab)
	if err != nil {
		return fmt.Errorf("failed to create reference of %s %s/%s. Reason: %v", ab.Kind, ab.Namespace, ab.Name, err)
	}

	targetAppGroup, targetAppResource := ab.AppGroupResource()
	prefix := targetAppResource
	if prefix == "" {
		prefix = ab.Kind
	}

	// if ab has backup annotations then ensure respective Repository and BackupConfiguration
	if meta_util.HasKey(ab.Annotations, api_v1beta1.KeyBackupBlueprint) {
		// backup annotations found. so, we have to ensure Repository and BackupConfiguration from BackupBlueprint
		verb, err := c.ensureAutoBackupResourcesForAppBinding(ab, targetAppGroup, targetAppResource)
		if err != nil {
			return c.handleAutoBackupResourcesCreationFailure(targetRef, err)
		}
		if verb != kutil.VerbUnchanged {
			return c.handleAutoBackupResourcesCreationSuccess(targetRef)
		}
	} else {
		// app binding does not have backup annotations. it might be removed or was never added.
		// if respective BackupConfiguration exist then backup annotations has been removed.
		// in this case, we have to remove the BackupConfiguration too.
		// however, we will keep Repository crd as it is required for restore.
		verb, err := c.ensureAutoBackupResourcesDeleted(targetRef, ab.Namespace, prefix)
		if err != nil {
			return c.handleAutoBackupResourcesDeletionFailure(targetRef, err)
		}
		if verb != kutil.VerbUnchanged {
			return c.handleAutoBackupResourcesDeletionSuccess(targetRef)
		}
	}
	return nil
}

func (c *StashController) ensureAutoBackupResourcesForAppBinding(ab *appCatalog.AppBinding, targetAppGroup, targetAppResource string) (kutil.VerbType, error) {

	backupBlueprintName, err := meta_util.GetStringValue(ab.Annotations, api_v1beta1.KeyBackupBlueprint)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	// resolve BackupBlueprint's variables
	inputs := make(map[string]string)
	inputs[apis.TargetAPIVersion] = ab.APIVersion
	inputs[apis.TargetKind] = strings.ToLower(ab.Kind)
	inputs[apis.TargetName] = ab.Name
	inputs[apis.TargetNamespace] = ab.Namespace
	inputs[apis.TargetAppVersion] = ab.Spec.Version
	inputs[apis.TargetAppGroup] = targetAppGroup
	inputs[apis.TargetAppResource] = targetAppResource
	inputs[apis.TargetAppType] = string(ab.Spec.Type)

	return c.ensureAutoBackupResources(backupBlueprintName, inputs, ab)
}
