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

package main

import (
	"reflect"

	crdv1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// CreateCRD creates CustomResourceDefinition
func CreateCRD(clientset apiextensionsclient.Interface) error {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdv1.VolumeSnapshotClassResourcePlural + "." + crdv1.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   crdv1.GroupName,
			Version: crdv1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.ClusterScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: crdv1.VolumeSnapshotClassResourcePlural,
				Kind:   reflect.TypeOf(crdv1.VolumeSnapshotClass{}).Name(),
			},
		},
	}
	res, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)

	if err != nil && !apierrors.IsAlreadyExists(err) {
		klog.Fatalf("failed to create VolumeSnapshotResource: %#v, err: %#v",
			res, err)
	}

	crd = &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdv1.VolumeSnapshotContentResourcePlural + "." + crdv1.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   crdv1.GroupName,
			Version: crdv1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.ClusterScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: crdv1.VolumeSnapshotContentResourcePlural,
				Kind:   reflect.TypeOf(crdv1.VolumeSnapshotContent{}).Name(),
			},
		},
	}
	res, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)

	if err != nil && !apierrors.IsAlreadyExists(err) {
		klog.Fatalf("failed to create VolumeSnapshotContentResource: %#v, err: %#v",
			res, err)
	}

	crd = &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdv1.VolumeSnapshotResourcePlural + "." + crdv1.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   crdv1.GroupName,
			Version: crdv1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: crdv1.VolumeSnapshotResourcePlural,
				Kind:   reflect.TypeOf(crdv1.VolumeSnapshot{}).Name(),
			},
		},
	}
	res, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)

	if err != nil && !apierrors.IsAlreadyExists(err) {
		klog.Fatalf("failed to create VolumeSnapshotResource: %#v, err: %#v",
			res, err)
	}

	return nil
}
