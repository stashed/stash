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

package types

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type URLMap struct {
	Scheme string
	Hosts  map[string]string
	Port   int
}

func NewURLMap(scheme string, port int) *URLMap {
	return &URLMap{
		Scheme: scheme,
		Hosts:  map[string]string{},
		Port:   port,
	}
}

func (um *URLMap) Insert(name, host string) {
	um.Hosts[name] = host
}

func (um *URLMap) Delete(hosts ...string) {
	for _, host := range hosts {
		delete(um.Hosts, host)
	}
}

func (um *URLMap) Has(host string) bool {
	_, contained := um.Hosts[host]
	return contained
}

func (um URLMap) Equal(s2 URLMap) bool {
	return um.Scheme == s2.Scheme &&
		um.Port == s2.Port &&
		reflect.DeepEqual(um.Hosts, s2.Hosts)
}

func (um *URLMap) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteRune('"')
	if um != nil {
		names := make([]string, 0, len(um.Hosts))
		for name := range um.Hosts {
			names = append(names, name)
		}
		sort.Strings(names)

		for i, name := range names {
			if i > 0 {
				b.WriteRune(',')
			}
			b.WriteString(name)
			b.WriteRune('=')
			b.WriteString(um.Scheme)
			b.WriteString("://")
			b.WriteString(um.Hosts[name])
			b.WriteString(":")
			b.WriteString(strconv.Itoa(um.Port))
		}
	}
	b.WriteRune('"')
	return []byte(b.String()), nil
}

func (um *URLMap) UnmarshalJSON(data []byte) error {
	if um == nil {
		return errors.New("jsontypes.URLMap: UnmarshalJSON on nil pointer")
	}

	n := len(data)
	if n < 2 {
		return fmt.Errorf("jsontypes.URLMap: UnmarshalJSON on invalid data %s", string(data))
	}
	if n == 2 && string(data) == `""` {
		return nil
	}
	um.Hosts = map[string]string{}

	entries := strings.Split(string(data[1:n-1]), ",")
	for _, entry := range entries {
		parts := strings.Split(entry, "=")
		if u, err := url.Parse(parts[1]); err == nil {
			um.Scheme = u.Scheme
			um.Hosts[parts[0]] = u.Hostname()
			um.Port, err = strconv.Atoi(u.Port())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
