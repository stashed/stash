/*
Copyright 2018 The Kubernetes Authors.

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

	crdv1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/util/slice"
	"os"
	"strconv"
	"time"
)

var (
	keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

type deprecatedSecretParamsMap struct {
	name                         string
	deprecatedSecretNameKey      string
	deprecatedSecretNamespaceKey string
	secretNameKey                string
	secretNamespaceKey           string
}

const (
	// CSI Parameters prefixed with csiParameterPrefix are not passed through
	// to the driver on CreateSnapshotRequest calls. Instead they are intended
	// to be used by the CSI external-snapshotter and maybe used to populate
	// fields in subsequent CSI calls or Kubernetes API objects.
	csiParameterPrefix = "csi.storage.k8s.io/"

	prefixedSnapshotterSecretNameKey      = csiParameterPrefix + "snapshotter-secret-name"
	prefixedSnapshotterSecretNamespaceKey = csiParameterPrefix + "snapshotter-secret-namespace"

	// [Deprecated] CSI Parameters that are put into fields but
	// NOT stripped from the parameters passed to CreateSnapshot
	snapshotterSecretNameKey      = "csiSnapshotterSecretName"
	snapshotterSecretNamespaceKey = "csiSnapshotterSecretNamespace"

	// Name of finalizer on VolumeSnapshotContents that are bound by VolumeSnapshots
	VolumeSnapshotContentFinalizer = "snapshot.storage.kubernetes.io/volumesnapshotcontent-protection"
	VolumeSnapshotFinalizer        = "snapshot.storage.kubernetes.io/volumesnapshot-protection"
)

var snapshotterSecretParams = deprecatedSecretParamsMap{
	name:                         "Snapshotter",
	deprecatedSecretNameKey:      snapshotterSecretNameKey,
	deprecatedSecretNamespaceKey: snapshotterSecretNamespaceKey,
	secretNameKey:                prefixedSnapshotterSecretNameKey,
	secretNamespaceKey:           prefixedSnapshotterSecretNamespaceKey,
}

func snapshotKey(vs *crdv1.VolumeSnapshot) string {
	return fmt.Sprintf("%s/%s", vs.Namespace, vs.Name)
}

func snapshotRefKey(vsref *v1.ObjectReference) string {
	return fmt.Sprintf("%s/%s", vsref.Namespace, vsref.Name)
}

// storeObjectUpdate updates given cache with a new object version from Informer
// callback (i.e. with events from etcd) or with an object modified by the
// controller itself. Returns "true", if the cache was updated, false if the
// object is an old version and should be ignored.
func storeObjectUpdate(store cache.Store, obj interface{}, className string) (bool, error) {
	objName, err := keyFunc(obj)
	if err != nil {
		return false, fmt.Errorf("Couldn't get key for object %+v: %v", obj, err)
	}
	oldObj, found, err := store.Get(obj)
	if err != nil {
		return false, fmt.Errorf("Error finding %s %q in controller cache: %v", className, objName, err)
	}

	objAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false, err
	}

	if !found {
		// This is a new object
		klog.V(4).Infof("storeObjectUpdate: adding %s %q, version %s", className, objName, objAccessor.GetResourceVersion())
		if err = store.Add(obj); err != nil {
			return false, fmt.Errorf("error adding %s %q to controller cache: %v", className, objName, err)
		}
		return true, nil
	}

	oldObjAccessor, err := meta.Accessor(oldObj)
	if err != nil {
		return false, err
	}

	objResourceVersion, err := strconv.ParseInt(objAccessor.GetResourceVersion(), 10, 64)
	if err != nil {
		return false, fmt.Errorf("error parsing ResourceVersion %q of %s %q: %s", objAccessor.GetResourceVersion(), className, objName, err)
	}
	oldObjResourceVersion, err := strconv.ParseInt(oldObjAccessor.GetResourceVersion(), 10, 64)
	if err != nil {
		return false, fmt.Errorf("error parsing old ResourceVersion %q of %s %q: %s", oldObjAccessor.GetResourceVersion(), className, objName, err)
	}

	// Throw away only older version, let the same version pass - we do want to
	// get periodic sync events.
	if oldObjResourceVersion > objResourceVersion {
		klog.V(4).Infof("storeObjectUpdate: ignoring %s %q version %s", className, objName, objAccessor.GetResourceVersion())
		return false, nil
	}

	klog.V(4).Infof("storeObjectUpdate updating %s %q with version %s", className, objName, objAccessor.GetResourceVersion())
	if err = store.Update(obj); err != nil {
		return false, fmt.Errorf("error updating %s %q in controller cache: %v", className, objName, err)
	}
	return true, nil
}

// GetSnapshotContentNameForSnapshot returns SnapshotContent.Name for the create VolumeSnapshotContent.
// The name must be unique.
func GetSnapshotContentNameForSnapshot(snapshot *crdv1.VolumeSnapshot) string {
	// If VolumeSnapshot object has SnapshotContentName, use it directly.
	// This might be the case for static provisioning.
	if len(snapshot.Spec.SnapshotContentName) > 0 {
		return snapshot.Spec.SnapshotContentName
	}
	// Construct SnapshotContentName for dynamic provisioning.
	return "snapcontent-" + string(snapshot.UID)
}

// IsDefaultAnnotation returns a boolean if
// the annotation is set
func IsDefaultAnnotation(obj metav1.ObjectMeta) bool {
	if obj.Annotations[IsDefaultSnapshotClassAnnotation] == "true" {
		return true
	}

	return false
}

// verifyAndGetSecretNameAndNamespaceTemplate gets the values (templates) associated
// with the parameters specified in "secret" and verifies that they are specified correctly.
func verifyAndGetSecretNameAndNamespaceTemplate(secret deprecatedSecretParamsMap, snapshotClassParams map[string]string) (nameTemplate, namespaceTemplate string, err error) {
	numName := 0
	numNamespace := 0
	if t, ok := snapshotClassParams[secret.deprecatedSecretNameKey]; ok {
		nameTemplate = t
		numName++
		klog.Warning(deprecationWarning(secret.deprecatedSecretNameKey, secret.secretNameKey, ""))
	}
	if t, ok := snapshotClassParams[secret.deprecatedSecretNamespaceKey]; ok {
		namespaceTemplate = t
		numNamespace++
		klog.Warning(deprecationWarning(secret.deprecatedSecretNamespaceKey, secret.secretNamespaceKey, ""))
	}
	if t, ok := snapshotClassParams[secret.secretNameKey]; ok {
		nameTemplate = t
		numName++
	}
	if t, ok := snapshotClassParams[secret.secretNamespaceKey]; ok {
		namespaceTemplate = t
		numNamespace++
	}

	if numName > 1 || numNamespace > 1 {
		// Double specified error
		return "", "", fmt.Errorf("%s secrets specified in paramaters with both \"csi\" and \"%s\" keys", secret.name, csiParameterPrefix)
	} else if numName != numNamespace {
		// Not both 0 or both 1
		return "", "", fmt.Errorf("either name and namespace for %s secrets specified, Both must be specified", secret.name)
	} else if numName == 1 {
		// Case where we've found a name and a namespace template
		if nameTemplate == "" || namespaceTemplate == "" {
			return "", "", fmt.Errorf("%s secrets specified in parameters but value of either namespace or name is empty", secret.name)
		}
		return nameTemplate, namespaceTemplate, nil
	} else if numName == 0 {
		// No secrets specified
		return "", "", nil
	} else {
		// THIS IS NOT A VALID CASE
		return "", "", fmt.Errorf("unknown error with getting secret name and namespace templates")
	}
}

// getSecretReference returns a reference to the secret specified in the given nameTemplate
//  and namespaceTemplate, or an error if the templates are not specified correctly.
// No lookup of the referenced secret is performed, and the secret may or may not exist.
//
// supported tokens for name resolution:
// - ${volumesnapshotcontent.name}
// - ${volumesnapshot.namespace}
// - ${volumesnapshot.name}
// - ${volumesnapshot.annotations['ANNOTATION_KEY']} (e.g. ${pvc.annotations['example.com/snapshot-create-secret-name']})
//
// supported tokens for namespace resolution:
// - ${volumesnapshotcontent.name}
// - ${volumesnapshot.namespace}
//
// an error is returned in the following situations:
// - the nameTemplate or namespaceTemplate contains a token that cannot be resolved
// - the resolved name is not a valid secret name
// - the resolved namespace is not a valid namespace name
func getSecretReference(snapshotClassParams map[string]string, snapContentName string, snapshot *crdv1.VolumeSnapshot) (*v1.SecretReference, error) {
	nameTemplate, namespaceTemplate, err := verifyAndGetSecretNameAndNamespaceTemplate(snapshotterSecretParams, snapshotClassParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get name and namespace template from params: %v", err)
	}

	if nameTemplate == "" && namespaceTemplate == "" {
		return nil, nil
	}

	ref := &v1.SecretReference{}

	// Secret namespace template can make use of the VolumeSnapshotContent name or the VolumeSnapshot namespace.
	// Note that neither of those things are under the control of the VolumeSnapshot user.
	namespaceParams := map[string]string{"volumesnapshotcontent.name": snapContentName}
	// snapshot may be nil when resolving create/delete snapshot secret names because the
	// snapshot may or may not exist at delete time
	if snapshot != nil {
		namespaceParams["volumesnapshot.namespace"] = snapshot.Namespace
	}

	resolvedNamespace, err := resolveTemplate(namespaceTemplate, namespaceParams)
	if err != nil {
		return nil, fmt.Errorf("error resolving value %q: %v", namespaceTemplate, err)
	}
	klog.V(4).Infof("GetSecretReference namespaceTemplate %s, namespaceParams: %+v, resolved %s", namespaceTemplate, namespaceParams, resolvedNamespace)

	if len(validation.IsDNS1123Label(resolvedNamespace)) > 0 {
		if namespaceTemplate != resolvedNamespace {
			return nil, fmt.Errorf("%q resolved to %q which is not a valid namespace name", namespaceTemplate, resolvedNamespace)
		}
		return nil, fmt.Errorf("%q is not a valid namespace name", namespaceTemplate)
	}
	ref.Namespace = resolvedNamespace

	// Secret name template can make use of the VolumeSnapshotContent name, VolumeSnapshot name or namespace,
	// or a VolumeSnapshot annotation.
	// Note that VolumeSnapshot name and annotations are under the VolumeSnapshot user's control.
	nameParams := map[string]string{"volumesnapshotcontent.name": snapContentName}
	if snapshot != nil {
		nameParams["volumesnapshot.name"] = snapshot.Name
		nameParams["volumesnapshot.namespace"] = snapshot.Namespace
		for k, v := range snapshot.Annotations {
			nameParams["volumesnapshot.annotations['"+k+"']"] = v
		}
	}
	resolvedName, err := resolveTemplate(nameTemplate, nameParams)
	if err != nil {
		return nil, fmt.Errorf("error resolving value %q: %v", nameTemplate, err)
	}
	if len(validation.IsDNS1123Subdomain(resolvedName)) > 0 {
		if nameTemplate != resolvedName {
			return nil, fmt.Errorf("%q resolved to %q which is not a valid secret name", nameTemplate, resolvedName)
		}
		return nil, fmt.Errorf("%q is not a valid secret name", nameTemplate)
	}
	ref.Name = resolvedName

	klog.V(4).Infof("GetSecretReference validated Secret: %+v", ref)
	return ref, nil
}

// resolveTemplate resolves the template by checking if the value is missing for a key
func resolveTemplate(template string, params map[string]string) (string, error) {
	missingParams := sets.NewString()
	resolved := os.Expand(template, func(k string) string {
		v, ok := params[k]
		if !ok {
			missingParams.Insert(k)
		}
		return v
	})
	if missingParams.Len() > 0 {
		return "", fmt.Errorf("invalid tokens: %q", missingParams.List())
	}
	return resolved, nil
}

// getCredentials retrieves credentials stored in v1.SecretReference
func getCredentials(k8s kubernetes.Interface, ref *v1.SecretReference) (map[string]string, error) {
	if ref == nil {
		return nil, nil
	}

	secret, err := k8s.CoreV1().Secrets(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s in namespace %s: %v", ref.Name, ref.Namespace, err)
	}

	credentials := map[string]string{}
	for key, value := range secret.Data {
		credentials[key] = string(value)
	}
	return credentials, nil
}

// NoResyncPeriodFunc Returns 0 for resyncPeriod in case resyncing is not needed.
func NoResyncPeriodFunc() time.Duration {
	return 0
}

// isContentDeletionCandidate checks if a volume snapshot content is a deletion candidate.
func isContentDeletionCandidate(content *crdv1.VolumeSnapshotContent) bool {
	return content.ObjectMeta.DeletionTimestamp != nil && slice.ContainsString(content.ObjectMeta.Finalizers, VolumeSnapshotContentFinalizer, nil)
}

// needToAddContentFinalizer checks if a Finalizer needs to be added for the volume snapshot content.
func needToAddContentFinalizer(content *crdv1.VolumeSnapshotContent) bool {
	return content.ObjectMeta.DeletionTimestamp == nil && !slice.ContainsString(content.ObjectMeta.Finalizers, VolumeSnapshotContentFinalizer, nil)
}

// isSnapshotDeletionCandidate checks if a volume snapshot is a deletion candidate.
func isSnapshotDeletionCandidate(snapshot *crdv1.VolumeSnapshot) bool {
	return snapshot.ObjectMeta.DeletionTimestamp != nil && slice.ContainsString(snapshot.ObjectMeta.Finalizers, VolumeSnapshotFinalizer, nil)
}

// needToAddSnapshotFinalizer checks if a Finalizer needs to be added for the volume snapshot.
func needToAddSnapshotFinalizer(snapshot *crdv1.VolumeSnapshot) bool {
	return snapshot.ObjectMeta.DeletionTimestamp == nil && !slice.ContainsString(snapshot.ObjectMeta.Finalizers, VolumeSnapshotFinalizer, nil)
}

func deprecationWarning(deprecatedParam, newParam, removalVersion string) string {
	if removalVersion == "" {
		removalVersion = "a future release"
	}
	newParamPhrase := ""
	if len(newParam) != 0 {
		newParamPhrase = fmt.Sprintf(", please use \"%s\" instead", newParam)
	}
	return fmt.Sprintf("\"%s\" is deprecated and will be removed in %s%s", deprecatedParam, removalVersion, newParamPhrase)
}

func removePrefixedParameters(param map[string]string) (map[string]string, error) {
	newParam := map[string]string{}
	for k, v := range param {
		if strings.HasPrefix(k, csiParameterPrefix) {
			// Check if its well known
			switch k {
			case prefixedSnapshotterSecretNameKey:
			case prefixedSnapshotterSecretNamespaceKey:
			default:
				return map[string]string{}, fmt.Errorf("found unknown parameter key \"%s\" with reserved namespace %s", k, csiParameterPrefix)
			}
		} else {
			// Don't strip, add this key-value to new map
			// Deprecated parameters prefixed with "csi" are not stripped to preserve backwards compatibility
			newParam[k] = v
		}
	}
	return newParam, nil
}
