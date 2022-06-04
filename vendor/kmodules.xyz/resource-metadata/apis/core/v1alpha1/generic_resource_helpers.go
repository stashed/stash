/*
Copyright AppsCode Inc. and Contributors.

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

package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strings"

	kmapi "kmodules.xyz/client-go/api/v1"
	mu "kmodules.xyz/client-go/meta"
	resourcemetrics "kmodules.xyz/resource-metrics"
	"kmodules.xyz/resource-metrics/api"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetGenericResourceName(item client.Object) string {
	return fmt.Sprintf("%s~%s", item.GetName(), item.GetObjectKind().GroupVersionKind().GroupKind())
}

func ParseGenericResourceName(name string) (string, schema.GroupKind, error) {
	parts := strings.SplitN(name, "~", 2)
	if len(parts) != 2 {
		return "", schema.GroupKind{}, fmt.Errorf("expected resource name %s in format {.metadata.name}~Kind.Group", name)
	}
	return parts[0], schema.ParseGroupKind(parts[1]), nil
}

func ToGenericResource(item client.Object, apiType *kmapi.ResourceID, cmeta *kmapi.ClusterMetadata) (*GenericResource, error) {
	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(item)
	if err != nil {
		return nil, err
	}

	s, err := status.Compute(&unstructured.Unstructured{
		Object: content,
	})
	if err != nil {
		return nil, err
	}

	// api.RegisteredTypes()

	var resstatus *runtime.RawExtension
	if v, ok, _ := unstructured.NestedFieldNoCopy(content, "status"); ok {
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert status to json, reason: %v", err)
		}
		resstatus = &runtime.RawExtension{Raw: data}
	}

	genres := GenericResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       ResourceKindGenericResource,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       GetGenericResourceName(item),
			GenerateName:               item.GetGenerateName(),
			Namespace:                  item.GetNamespace(),
			SelfLink:                   "",
			UID:                        "gres-" + item.GetUID(),
			ResourceVersion:            item.GetResourceVersion(),
			Generation:                 item.GetGeneration(),
			CreationTimestamp:          item.GetCreationTimestamp(),
			DeletionTimestamp:          item.GetDeletionTimestamp(),
			DeletionGracePeriodSeconds: item.GetDeletionGracePeriodSeconds(),
			Labels:                     item.GetLabels(),
			Annotations:                map[string]string{},
			// OwnerReferences:            item.GetOwnerReferences(),
			// Finalizers:                 item.GetFinalizers(),
			// ClusterName: item.GetClusterName(),
			// ManagedFields:              nil,
		},
		Spec: GenericResourceSpec{
			Cluster:              *cmeta,
			APIType:              *apiType,
			Name:                 item.GetName(),
			UID:                  item.GetUID(),
			Replicas:             0,
			RoleReplicas:         nil,
			Mode:                 "",
			TotalResource:        core.ResourceRequirements{},
			AppResource:          core.ResourceRequirements{},
			RoleResourceLimits:   nil,
			RoleResourceRequests: nil,

			Status: GenericResourceStatus{
				Status:  s.Status.String(),
				Message: s.Message,
			},
		},
		Status: resstatus,
	}
	for k, v := range item.GetAnnotations() {
		if k != mu.LastAppliedConfigAnnotation {
			genres.Annotations[k] = v
		}
	}
	{
		if v, ok, _ := unstructured.NestedString(content, "spec", "version"); ok {
			genres.Spec.Version = v
		}
	}
	if api.IsRegistered(apiType.GroupVersionKind()) {
		{
			rv, err := resourcemetrics.Replicas(content)
			if err != nil {
				return nil, err
			}
			genres.Spec.Replicas = rv
		}
		{
			rv, err := resourcemetrics.RoleReplicas(content)
			if err != nil {
				return nil, err
			}
			genres.Spec.RoleReplicas = rv
		}
		{
			rv, err := resourcemetrics.Mode(content)
			if err != nil {
				return nil, err
			}
			genres.Spec.Mode = rv
		}
		{
			rv, err := resourcemetrics.TotalResourceRequests(content)
			if err != nil {
				return nil, err
			}
			genres.Spec.TotalResource.Requests = rv
		}
		{
			rv, err := resourcemetrics.TotalResourceLimits(content)
			if err != nil {
				return nil, err
			}
			genres.Spec.TotalResource.Limits = rv
		}
		{
			rv, err := resourcemetrics.AppResourceRequests(content)
			if err != nil {
				return nil, err
			}
			genres.Spec.AppResource.Requests = rv
		}
		{
			rv, err := resourcemetrics.AppResourceLimits(content)
			if err != nil {
				return nil, err
			}
			genres.Spec.AppResource.Limits = rv
		}
		{
			rv, err := resourcemetrics.RoleResourceRequests(content)
			if err != nil {
				return nil, err
			}
			genres.Spec.RoleResourceRequests = rv
		}
		{
			rv, err := resourcemetrics.RoleResourceLimits(content)
			if err != nil {
				return nil, err
			}
			genres.Spec.RoleResourceLimits = rv
		}
	}
	return &genres, nil
}
