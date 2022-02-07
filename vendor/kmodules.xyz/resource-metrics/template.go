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
	"html/template"
	"reflect"
	ttemplate "text/template"

	"kmodules.xyz/resource-metrics/api"

	"gomodules.xyz/encoding/json"
	core "k8s.io/api/core/v1"
)

// TxtFuncMap returns a 'text/template'.FuncMap
func TxtFuncMap() ttemplate.FuncMap {
	return ttemplate.FuncMap(GenericFuncMap())
}

// HtmlFuncMap returns an 'html/template'.Funcmap
func HtmlFuncMap() template.FuncMap {
	return template.FuncMap(GenericFuncMap())
}

// GenericFuncMap returns a copy of the basic function map as a map[string]interface{}.
func GenericFuncMap() map[string]interface{} {
	return map[string]interface{}{
		"k8s_resource_replicas":          tplReplicaFn,
		"k8s_resource_replicas_by_roles": tplRoleReplicaFn,
		"k8s_resource_mode":              tplModeFn,
		"k8s_resource_uses_tls":          tplUsesTLSFn,
		"k8s_total_resource_limits":      tplTotalResourceLimitsFn,
		"k8s_total_resource_requests":    tplTotalResourceRequestsFn,
		"k8s_app_resource_limits":        tplAppResourceLimitsFn,
		"k8s_app_resource_requests":      tplAppResourceRequestsFn,
	}
}

func tplReplicaFn(data interface{}) (int64, error) {
	obj, err := toObject(data)
	if err != nil {
		return 0, err
	}

	c, err := api.Load(obj)
	if err != nil {
		return 0, err
	}
	return c.Replicas(obj)
}

func tplRoleReplicaFn(data interface{}) (interface{}, error) {
	obj, err := toObject(data)
	if err != nil {
		return nil, err
	}

	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	replicaList, err := c.RoleReplicas(obj)
	if err != nil {
		return nil, err
	}

	has := func(role api.PodRole) bool {
		_, ok := replicaList[role]
		return ok
	}
	if has(api.PodRoleDefault) {
		if len(replicaList) == 1 {
			return replicaList[api.PodRoleDefault], nil
		} else {
			// case: elasticsearch
			delete(replicaList, api.PodRoleDefault)
			return replicaList, nil
		}
	} else if has(api.PodRoleTotalShard) {
		// case: mongodb, redis
		delete(replicaList, api.PodRoleTotalShard)
		return replicaList, nil
	}
	return replicaList, nil
}

func tplModeFn(data interface{}) (string, error) {
	obj, err := toObject(data)
	if err != nil {
		return "", err
	}

	c, err := api.Load(obj)
	if err != nil {
		return "", err
	}
	return c.Mode(obj)
}

func tplUsesTLSFn(data interface{}) (bool, error) {
	obj, err := toObject(data)
	if err != nil {
		return false, err
	}

	c, err := api.Load(obj)
	if err != nil {
		return false, err
	}
	return c.UsesTLS(obj)
}

func tplTotalResourceLimitsFn(data interface{}) (core.ResourceList, error) {
	obj, err := toObject(data)
	if err != nil {
		return nil, err
	}

	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.TotalResourceLimits(obj)
}

func tplTotalResourceRequestsFn(data interface{}) (core.ResourceList, error) {
	obj, err := toObject(data)
	if err != nil {
		return nil, err
	}

	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.TotalResourceRequests(obj)
}

func tplAppResourceLimitsFn(data interface{}) (core.ResourceList, error) {
	obj, err := toObject(data)
	if err != nil {
		return nil, err
	}

	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.AppResourceLimits(obj)
}

func tplAppResourceRequestsFn(data interface{}) (core.ResourceList, error) {
	obj, err := toObject(data)
	if err != nil {
		return nil, err
	}

	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.AppResourceRequests(obj)
}

func toObject(data interface{}) (map[string]interface{}, error) {
	var obj map[string]interface{}
	if v, ok := data.(map[string]interface{}); ok {
		obj = v
	} else if str, ok := data.(string); ok {
		err := json.Unmarshal([]byte(str), &obj)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown obj type %v", reflect.TypeOf(data).String())
	}
	return obj, nil
}
