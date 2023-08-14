/*
Copyright AppsCode Inc. and Contributors

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

package apiextensions

import (
	"context"
	"fmt"
	"time"

	v1 "kmodules.xyz/client-go/apiextensions/v1"
	meta_util "kmodules.xyz/client-go/meta"

	"github.com/pkg/errors"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

func RegisterCRDs(client crd_cs.Interface, crds []*CustomResourceDefinition) error {
	for _, crd := range crds {
		// Use crd v1 for k8s >= 1.16, if available
		// ref: https://github.com/kubernetes/kubernetes/issues/91395
		if crd.V1 == nil {
			gvr := schema.GroupVersionResource{
				Group:    crd.V1beta1.Spec.Group,
				Version:  crd.V1beta1.Spec.Versions[0].Name,
				Resource: crd.V1beta1.Spec.Names.Plural,
			}
			return fmt.Errorf("missing V1 definition for %s", gvr)
		}
		_, _, err := v1.CreateOrUpdateCustomResourceDefinition(
			context.TODO(),
			client,
			crd.V1.Name,
			func(in *crdv1.CustomResourceDefinition) *crdv1.CustomResourceDefinition {
				in.Labels = meta_util.OverwriteKeys(in.Labels, crd.V1.Labels)
				in.Annotations = meta_util.OverwriteKeys(in.Annotations, crd.V1.Annotations)

				in.Spec = crd.V1.Spec
				return in
			},
			metav1.UpdateOptions{},
		)
		if err != nil && !kerr.IsAlreadyExists(err) {
			return err
		}
	}
	return WaitForCRDReady(client, crds)
}

func WaitForCRDReady(client crd_cs.Interface, crds []*CustomResourceDefinition) error {
	err := wait.Poll(3*time.Second, 5*time.Minute, func() (bool, error) {
		for _, crd := range crds {
			var gvr schema.GroupVersionResource
			if crd.V1 != nil {
				gvr = schema.GroupVersionResource{
					Group:    crd.V1.Spec.Group,
					Version:  crd.V1.Spec.Versions[0].Name,
					Resource: crd.V1.Spec.Names.Plural,
				}
			} else if crd.V1beta1 != nil {
				gvr = schema.GroupVersionResource{
					Group:    crd.V1beta1.Spec.Group,
					Version:  crd.V1beta1.Spec.Versions[0].Name,
					Resource: crd.V1beta1.Spec.Names.Plural,
				}
			}

			objc, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), gvr.GroupResource().String(), metav1.GetOptions{})
			if err != nil {
				if kerr.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}

			for _, c := range objc.Status.Conditions {
				if c.Type == "NamesAccepted" && c.Status == crdv1.ConditionFalse {
					return false, fmt.Errorf("CRD %s %s: %s", gvr.GroupResource(), c.Reason, c.Message)
				}
				if c.Type == "Established" {
					if c.Status == crdv1.ConditionFalse && c.Reason != "Installing" {
						return false, fmt.Errorf("CRD %s %s: %s", gvr.GroupResource(), c.Reason, c.Message)
					}
					if c.Status == crdv1.ConditionTrue {
						break
					}
				}
			}
		}
		return true, nil
	})
	return errors.Wrap(err, "timed out waiting for CRD")
}
