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

package meta

import (
	"strconv"
	"time"

	kutil "kmodules.xyz/client-go"
)

type ParserFunc func(map[string]string, string) (interface{}, error)

var (
	_ ParserFunc = GetBool
	_ ParserFunc = GetInt
	_ ParserFunc = GetString
	_ ParserFunc = GetList
	_ ParserFunc = GetMap
	_ ParserFunc = GetFloat
	_ ParserFunc = GetDuration
)

func GetBool(m map[string]string, key string) (interface{}, error) {
	if m == nil {
		return false, kutil.ErrNotFound
	}
	v, ok := m[key]
	if !ok {
		return false, kutil.ErrNotFound
	}
	return strconv.ParseBool(v)
}

func GetBoolValue(m map[string]string, key string) (bool, error) {
	v, err := GetBool(m, key)
	return v.(bool), err
}

func GetInt(m map[string]string, key string) (interface{}, error) {
	if m == nil {
		return 0, kutil.ErrNotFound
	}
	v, ok := m[key]
	if !ok {
		return 0, kutil.ErrNotFound
	}
	return strconv.Atoi(v)
}

func GetIntValue(m map[string]string, key string) (int, error) {
	v, err := GetInt(m, key)
	return v.(int), err
}

func GetString(m map[string]string, key string) (interface{}, error) {
	if m == nil {
		return "", kutil.ErrNotFound
	}
	v, ok := m[key]
	if !ok {
		return "", kutil.ErrNotFound
	}
	return v, nil
}

func GetStringValue(m map[string]string, key string) (string, error) {
	v, err := GetString(m, key)
	return v.(string), err
}

func HasKey(m map[string]string, key string) bool {
	if m == nil {
		return false
	}
	_, ok := m[key]
	return ok
}

func RemoveKey(m map[string]string, key string) map[string]string {
	if m == nil {
		return nil
	}
	delete(m, key)
	return m
}

func GetList(m map[string]string, key string) (interface{}, error) {
	if m == nil {
		return []string{}, kutil.ErrNotFound
	}
	s, ok := m[key]
	if !ok {
		return []string{}, kutil.ErrNotFound
	}
	v := make([]string, 0)
	err := json.Unmarshal([]byte(s), &v)
	return v, err
}

func GetListValue(m map[string]string, key string) ([]string, error) {
	v, err := GetList(m, key)
	return v.([]string), err
}

func GetMap(m map[string]string, key string) (interface{}, error) {
	if m == nil {
		return map[string]string{}, kutil.ErrNotFound
	}
	s, ok := m[key]
	if !ok {
		return map[string]string{}, kutil.ErrNotFound
	}
	v := make(map[string]string)
	err := json.Unmarshal([]byte(s), &v)
	return v, err
}

func GetMapValue(m map[string]string, key string) (map[string]string, error) {
	v, err := GetMap(m, key)
	return v.(map[string]string), err
}

func GetFloat(m map[string]string, key string) (interface{}, error) {
	if m == nil {
		return 0.0, kutil.ErrNotFound
	}
	f, ok := m[key]
	if !ok {
		return 0.0, kutil.ErrNotFound
	}

	return strconv.ParseFloat(f, 64)
}

func GetFloatValue(m map[string]string, key string) (float64, error) {
	v, err := GetFloat(m, key)
	return v.(float64), err
}

func GetDuration(m map[string]string, key string) (interface{}, error) {
	if m == nil {
		return time.Duration(0), kutil.ErrNotFound
	}
	d, ok := m[key]
	if !ok {
		return time.Duration(0), kutil.ErrNotFound
	}

	return time.ParseDuration(d)
}

func GetDurationValue(m map[string]string, key string) (time.Duration, error) {
	v, err := GetDuration(m, key)
	return v.(time.Duration), err
}

type GetFunc func(map[string]string) (interface{}, error)

func ParseFor(key string, fn ParserFunc) GetFunc {
	return func(m map[string]string) (interface{}, error) {
		return fn(m, key)
	}
}

func GetStringValueForKeys(m map[string]string, key string, alts ...string) (string, error) {
	if m == nil {
		return "", kutil.ErrNotFound
	}
	keys := append([]string{key}, alts...)
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v, nil
		}
	}
	return "", kutil.ErrNotFound
}

func GetBytesForKeys(m map[string][]byte, key string, alts ...string) ([]byte, error) {
	if m == nil {
		return nil, kutil.ErrNotFound
	}
	keys := append([]string{key}, alts...)
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v, nil
		}
	}
	return nil, kutil.ErrNotFound
}
