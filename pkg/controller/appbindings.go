package controller

import (
	"fmt"
	"strings"

	kutil "kmodules.xyz/client-go"

	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/queue"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	catalog_scheme "kmodules.xyz/custom-resources/client/clientset/versioned/scheme"
	"stash.appscode.dev/stash/apis"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/resolve"
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

		err := c.applyBackupAnnotationLogicForAppBinding(ab)
		if err != nil {
			return err
		}
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
		verb, err := c.ensureAutoBackupResourcesForAppBinding(ab, targetRef, targetAppGroup, targetAppResource, prefix)
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
		verb, err := c.ensureAutoBackupResourcesForAppBindingDeleted(ab, targetRef, prefix)
		if err != nil {
			return c.handleAutoBackupResourcesDeletionFailure(targetRef, err)
		}
		if verb != kutil.VerbUnchanged {
			return c.handleAutoBackupResourcesDeletionSuccess(targetRef)
		}
	}
	return nil
}

func (c *StashController) ensureAutoBackupResourcesForAppBinding(
	ab *appCatalog.AppBinding,
	targetRef *core.ObjectReference,
	targetAppGroup string,
	targetAppResource string,
	prefix string,
) (kutil.VerbType, error) {

	backupBlueprintName, err := meta_util.GetStringValue(ab.Annotations, api_v1beta1.KeyBackupBlueprint)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	backupBlueprint, err := c.stashClient.StashV1beta1().BackupBlueprints().Get(backupBlueprintName, metav1.GetOptions{})
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

	err = resolve.ResolveBackupBlueprint(backupBlueprint, inputs)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	// ensure Repository crd
	verb1, err := c.ensureRepository(backupBlueprint, targetRef, prefix)
	if err != nil {
		return kutil.VerbUnchanged, err
	}

	// ensure BackupConfiguration crd
	verb2, err := c.ensureBackupConfiguration(backupBlueprint, nil, nil, targetRef, prefix)
	if err != nil {
		return kutil.VerbUnchanged, err
	}
	// if both of the verb is unchanged then no create/update happened to the auto backup resources
	if verb1 == kutil.VerbUnchanged || verb2 == kutil.VerbUnchanged {
		return kutil.VerbUnchanged, nil
	}
	// auto backup resources has been created/updated
	// we will use this information to write event to AppBinding
	// so, "created" or "updated" verb has same effect to the end result
	// we can return any of them.
	return kutil.VerbCreated, nil
}

func (c *StashController) ensureAutoBackupResourcesForAppBindingDeleted(
	ab *appCatalog.AppBinding,
	targetRef *core.ObjectReference,
	prefix string,
) (kutil.VerbType, error) {

	_, err := c.stashClient.StashV1beta1().BackupConfigurations(ab.Namespace).Get(getBackupConfigurationName(targetRef, prefix), metav1.GetOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return kutil.VerbUnchanged, err
	}
	// BackupConfiguration exist. so, we have to remove it.
	err = c.stashClient.StashV1beta1().BackupConfigurations(ab.Namespace).Delete(getBackupConfigurationName(targetRef, prefix), meta_util.DeleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return kutil.VerbUnchanged, err
	}
	return kutil.VerbDeleted, nil
}
