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

package yaml

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

// ref: https://github.com/kubernetes/apimachinery/blob/48159c651603a061d16fa1dbab2cfe32eceba27a/pkg/apis/meta/v1/unstructured/helpers.go

// NestedFieldCopy returns a deep copy of the value of a nested field.
// Returns false if the value is missing.
// No error is returned for a nil field.
//
// Note: fields passed to this function are treated as keys within the passed
// object; no array/slice syntax is supported.
func NestedFieldCopy(obj yaml.MapSlice, fields ...string) (interface{}, bool, error) {
	val, found, err := NestedFieldNoCopy(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}
	return DeepCopyYAMLValue(val), true, nil
}

// NestedFieldNoCopy returns a reference to a nested field.
// Returns false if value is not found and an error if unable
// to traverse obj.
//
// Note: fields passed to this function are treated as keys within the passed
// object; no array/slice syntax is supported.
func NestedFieldNoCopy(obj yaml.MapSlice, fields ...string) (interface{}, bool, error) {
	var val interface{} = obj

	for i, field := range fields {
		if val == nil {
			return nil, false, nil
		}
		if m, ok := val.(yaml.MapSlice); ok {
			val, ok = _get(m, field)
			if !ok {
				return nil, false, nil
			}
		} else {
			return nil, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected yaml.MapSlice", jsonPath(fields[:i+1]), val, val)
		}
	}
	return val, true, nil
}

// NestedString returns the string value of a nested field.
// Returns false if value is not found and an error if not a string.
func NestedString(obj yaml.MapSlice, fields ...string) (string, bool, error) {
	val, found, err := NestedFieldNoCopy(obj, fields...)
	if !found || err != nil {
		return "", found, err
	}
	s, ok := val.(string)
	if !ok {
		return "", false, fmt.Errorf("%v accessor error: %v is of the type %T, expected string", jsonPath(fields), val, val)
	}
	return s, true, nil
}

// NestedBool returns the bool value of a nested field.
// Returns false if value is not found and an error if not a bool.
func NestedBool(obj yaml.MapSlice, fields ...string) (bool, bool, error) {
	val, found, err := NestedFieldNoCopy(obj, fields...)
	if !found || err != nil {
		return false, found, err
	}
	b, ok := val.(bool)
	if !ok {
		return false, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected bool", jsonPath(fields), val, val)
	}
	return b, true, nil
}

// NestedFloat64 returns the float64 value of a nested field.
// Returns false if value is not found and an error if not a float64.
func NestedFloat64(obj yaml.MapSlice, fields ...string) (float64, bool, error) {
	val, found, err := NestedFieldNoCopy(obj, fields...)
	if !found || err != nil {
		return 0, found, err
	}
	f, ok := val.(float64)
	if !ok {
		return 0, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected float64", jsonPath(fields), val, val)
	}
	return f, true, nil
}

// NestedInt64 returns the int64 value of a nested field.
// Returns false if value is not found and an error if not an int64.
func NestedInt64(obj yaml.MapSlice, fields ...string) (int64, bool, error) {
	val, found, err := NestedFieldNoCopy(obj, fields...)
	if !found || err != nil {
		return 0, found, err
	}
	i, ok := val.(int64)
	if !ok {
		return 0, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected int64", jsonPath(fields), val, val)
	}
	return i, true, nil
}

// NestedStringSlice returns a copy of []string value of a nested field.
// Returns false if value is not found and an error if not a []interface{} or contains non-string items in the slice.
func NestedStringSlice(obj yaml.MapSlice, fields ...string) ([]string, bool, error) {
	val, found, err := NestedFieldNoCopy(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}
	m, ok := val.([]interface{})
	if !ok {
		return nil, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected []interface{}", jsonPath(fields), val, val)
	}
	strSlice := make([]string, 0, len(m))
	for _, v := range m {
		if str, ok := v.(string); ok {
			strSlice = append(strSlice, str)
		} else {
			return nil, false, fmt.Errorf("%v accessor error: contains non-string key in the slice: %v is of the type %T, expected string", jsonPath(fields), v, v)
		}
	}
	return strSlice, true, nil
}

// NestedSlice returns a deep copy of []interface{} value of a nested field.
// Returns false if value is not found and an error if not a []interface{}.
func NestedSlice(obj yaml.MapSlice, fields ...string) ([]interface{}, bool, error) {
	val, found, err := NestedFieldNoCopy(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}
	_, ok := val.([]interface{})
	if !ok {
		return nil, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected []interface{}", jsonPath(fields), val, val)
	}
	return DeepCopyYAMLValue(val).([]interface{}), true, nil
}

// NestedStringMap returns a copy of map[string]string value of a nested field.
// Returns false if value is not found and an error if not a yaml.MapSlice or contains non-string values in the map.
func NestedStringMap(obj yaml.MapSlice, fields ...string) (map[string]string, bool, error) {
	m, found, err := nestedMapNoCopy(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}
	strMap := make(map[string]string, len(m))
	for _, e := range m {
		if str, ok := e.Value.(string); ok {
			strMap[e.Key.(string)] = str
		} else {
			return nil, false, fmt.Errorf("%v accessor error: contains non-string key in the map: %v is of the type %T, expected string", jsonPath(fields), e.Value, e.Value)
		}
	}
	return strMap, true, nil
}

// NestedMap returns a deep copy of yaml.MapSlice value of a nested field.
// Returns false if value is not found and an error if not a yaml.MapSlice.
func NestedMap(obj yaml.MapSlice, fields ...string) (yaml.MapSlice, bool, error) {
	m, found, err := nestedMapNoCopy(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}
	return DeepCopyYAML(m), true, nil
}

// nestedMapNoCopy returns a yaml.MapSlice value of a nested field.
// Returns false if value is not found and an error if not a yaml.MapSlice.
func nestedMapNoCopy(obj yaml.MapSlice, fields ...string) (yaml.MapSlice, bool, error) {
	val, found, err := NestedFieldNoCopy(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}
	m, ok := val.(yaml.MapSlice)
	if !ok {
		return nil, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected yaml.MapSlice", jsonPath(fields), val, val)
	}
	return m, true, nil
}

// SetNestedField sets the value of a nested field to a deep copy of the value provided.
// Returns an error if value cannot be set because one of the nesting levels is not a yaml.MapSlice.
func SetNestedField(obj *yaml.MapSlice, value interface{}, fields ...string) error {
	return setNestedFieldNoCopy(obj, DeepCopyYAMLValue(value), fields, fields...)
}

func setNestedFieldNoCopy(obj *yaml.MapSlice, value interface{}, fp []string, fields ...string) error {
	for i, item := range *obj {
		if item.Key.(string) == fields[0] {
			if len(fields) == 1 {
				(*obj)[i].Value = value
				return nil
			} else {
				if v, ok := item.Value.(yaml.MapSlice); ok {
					err := setNestedFieldNoCopy(&v, value, fp, fields[1:]...)
					if err != nil {
						return err
					}
					(*obj)[i].Value = v
					return nil
				} else {
					return fmt.Errorf("value cannot be set because %v is not a yaml.MapSlice", jsonPath(fp[:len(fp)-len(fields)+1]))
				}
			}
		}
	}

	if len(fields) == 1 {
		*obj = append(*obj, yaml.MapItem{
			Key:   fields[0],
			Value: value,
		})
	} else {
		var newVal yaml.MapSlice
		err := setNestedFieldNoCopy(&newVal, value, fp, fields[1:]...)
		if err != nil {
			return err
		}
		*obj = append(*obj, yaml.MapItem{
			Key:   fields[0],
			Value: newVal,
		})
	}
	return nil
}

func _get(ms yaml.MapSlice, field string) (interface{}, bool) {
	for _, item := range ms {
		if item.Key.(string) == field {
			return item.Value, true
		}
	}
	return nil, false
}

func _entry(ms yaml.MapSlice, field string) interface{} {
	v, _ := _get(ms, field)
	return v
}

func _set(ms *yaml.MapSlice, field string, v interface{}) {
	for i, item := range *ms {
		if item.Key.(string) == field {
			(*ms)[i].Value = v
			return
		}
	}
	*ms = append(*ms, yaml.MapItem{
		Key:   field,
		Value: v,
	})
}

func _delete(ms *yaml.MapSlice, field string) {
	for i, item := range *ms {
		if item.Key.(string) == field {
			*ms = append((*ms)[:i], (*ms)[i+1:]...)
			return
		}
	}
}

// SetNestedStringSlice sets the string slice value of a nested field.
// Returns an error if value cannot be set because one of the nesting levels is not a yaml.MapSlice.
func SetNestedStringSlice(obj *yaml.MapSlice, value []string, fields ...string) error {
	m := make([]interface{}, 0, len(value)) // convert []string into []interface{}
	for _, v := range value {
		m = append(m, v)
	}
	return setNestedFieldNoCopy(obj, m, fields, fields...)
}

// SetNestedSlice sets the slice value of a nested field.
// Returns an error if value cannot be set because one of the nesting levels is not a yaml.MapSlice.
func SetNestedSlice(obj *yaml.MapSlice, value []interface{}, fields ...string) error {
	return SetNestedField(obj, value, fields...)
}

// SetNestedStringMap sets the map[string]string value of a nested field.
// Returns an error if value cannot be set because one of the nesting levels is not a yaml.MapSlice.
func SetNestedStringMap(obj *yaml.MapSlice, value map[string]string, fields ...string) error {
	var m yaml.MapSlice // convert map[string]string into yaml.MapSlice
	for k, v := range value {
		_set(&m, k, v)
	}
	return setNestedFieldNoCopy(obj, m, fields, fields...)
}

// SetNestedMap sets the yaml.MapSlice value of a nested field.
// Returns an error if value cannot be set because one of the nesting levels is not a yaml.MapSlice.
func SetNestedMap(obj *yaml.MapSlice, value yaml.MapSlice, fields ...string) error {
	return SetNestedField(obj, value, fields...)
}

// RemoveNestedField removes the nested field from the obj.
func RemoveNestedField(obj *yaml.MapSlice, fields ...string) {
	for i, item := range *obj {
		if item.Key.(string) == fields[0] {
			if len(fields) == 1 {
				*obj = append((*obj)[:i], (*obj)[i+1:]...)
			} else {
				if v, ok := item.Value.(yaml.MapSlice); ok {
					RemoveNestedField(&v, fields[1:]...)
					(*obj)[i].Value = v
				}
			}
			return
		}
	}
}

func jsonPath(fields []string) string {
	return "." + strings.Join(fields, ".")
}

// DeepCopyYAML deep copies the passed value, assuming it is a valid JSON representation i.e. only contains
// types produced by json.Unmarshal() and also int64.
// bool, int64, float64, string, []interface{}, map[string]interface{}, json.Number and nil
func DeepCopyYAML(x yaml.MapSlice) yaml.MapSlice {
	return DeepCopyYAMLValue(x).(yaml.MapSlice)
}

// DeepCopyYAMLValue deep copies the passed value, assuming it is a valid JSON representation i.e. only contains
// types produced by json.Unmarshal() and also int64.
// bool, int64, float64, string, []interface{}, map[string]interface{}, json.Number and nil
func DeepCopyYAMLValue(x interface{}) interface{} {
	switch x := x.(type) {
	case map[string]interface{}:
		if x == nil {
			// Typed nil - an interface{} that contains a type map[string]interface{} with a value of nil
			return x
		}
		clone := make(map[string]interface{}, len(x))
		for k, v := range x {
			clone[k] = DeepCopyYAMLValue(v)
		}
		return clone
	case yaml.MapSlice:
		if x == nil {
			// Typed nil - an interface{} that contains a type map[string]interface{} with a value of nil
			return x
		}
		clone := make(yaml.MapSlice, len(x))
		for i, e := range x {
			clone[i] = yaml.MapItem{
				Key:   DeepCopyYAMLValue(e.Key),
				Value: DeepCopyYAMLValue(e.Value),
			}
		}
		return clone
	case []interface{}:
		if x == nil {
			// Typed nil - an interface{} that contains a type []interface{} with a value of nil
			return x
		}
		clone := make([]interface{}, len(x))
		for i, v := range x {
			clone[i] = DeepCopyYAMLValue(v)
		}
		return clone
	case string, int8, uint8, int16, uint16, int32, uint32, int64, uint64, int, uint, float32, float64, bool, nil, json.Number:
		return x
	default:
		panic(fmt.Errorf("cannot deep copy %T", x))
	}
}
