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

package resourcemetrics

import (
	"fmt"
	"reflect"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// EvalFuncs for https://github.com/gomodules/eval
func EvalFuncs() map[string]func(arguments ...interface{}) (interface{}, error) {
	return map[string]func(arguments ...interface{}) (interface{}, error){
		"resource_replicas":       resourceReplicas,
		"resource_mode":           resourceMode,
		"resource_uses_tls":       resourceUsesTLS,
		"total_resource_limits":   totalResourceLimits,
		"total_resource_requests": totalResourceRequests,
		"app_resource_limits":     appResourceLimits,
		"app_resource_requests":   appResourceRequests,
	}
}

// resourceReplicas(resource_obj)
func resourceReplicas(args ...interface{}) (interface{}, error) {
	return Replicas(args[0].(map[string]interface{}))
}

// resourceMode(resource_obj)
func resourceMode(args ...interface{}) (interface{}, error) {
	return Mode(args[0].(map[string]interface{}))
}

// resourceUsesTLS(resource_obj)
func resourceUsesTLS(args ...interface{}) (interface{}, error) {
	return UsesTLS(args[0].(map[string]interface{}))
}

// totalResourceLimits(resource_obj, resource_type) => cpu cores (float64)
func totalResourceLimits(args ...interface{}) (interface{}, error) {
	rr, err := TotalResourceLimits(args[0].(map[string]interface{}))
	if err != nil {
		return nil, err
	}
	return resourceQuantity(rr, args[1])
}

// totalResourceRequests(resource_obj, resource_type)
func totalResourceRequests(args ...interface{}) (interface{}, error) {
	rr, err := TotalResourceRequests(args[0].(map[string]interface{}))
	if err != nil {
		return nil, err
	}
	return resourceQuantity(rr, args[1])
}

// appResourceLimits(resource_obj, resource_type)
func appResourceLimits(args ...interface{}) (interface{}, error) {
	rr, err := AppResourceLimits(args[0].(map[string]interface{}))
	if err != nil {
		return nil, err
	}
	return resourceQuantity(rr, args[1])
}

// appResourceRequests(resource_obj, resource_type)
func appResourceRequests(args ...interface{}) (interface{}, error) {
	rr, err := AppResourceRequests(args[0].(map[string]interface{}))
	if err != nil {
		return nil, err
	}
	return resourceQuantity(rr, args[1])
}

func resourceQuantity(rr core.ResourceList, resourceName interface{}) (interface{}, error) {
	var name core.ResourceName

	switch u := resourceName.(type) {
	case core.ResourceName:
		name = u
	case string:
		name = core.ResourceName(u)
	default:
		return nil, fmt.Errorf("ResourceName %v with unrecognized type %v", resourceName, reflect.TypeOf(resourceName))
	}

	q := rr[name]
	return resourceQuantityAsFloat64(name, q), nil
}

func resourceQuantityAsFloat64(name core.ResourceName, q resource.Quantity) float64 {
	// WARNING: q.AsApproximateFloat64() does not work
	// ref: https://github.com/kubernetes/kube-state-metrics/blob/16e8f54c9e7f9f4b4ad73002e03e9d0dcee5b1ce/internal/store/pod.go#L234
	switch name {
	case core.ResourceCPU:
		return float64(q.MilliValue()) / 1000
	default:
		return float64(q.Value())
	}
}
