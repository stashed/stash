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
	"sort"
	"strconv"
	"strings"

	"gomodules.xyz/sets"
)

type URLSet struct {
	Scheme string
	Hosts  sets.String
	Port   int
}

func NewURLSet(scheme string, port int) *URLSet {
	return &URLSet{
		Scheme: scheme,
		Hosts:  sets.NewString(),
		Port:   port,
	}
}

func (us *URLSet) Insert(hosts ...string) {
	us.Hosts.Insert(hosts...)
}

func (us *URLSet) Delete(hosts ...string) {
	us.Hosts.Delete(hosts...)
}

func (us *URLSet) Has(host string) bool {
	return us.Hosts.Has(host)
}

func (s1 URLSet) Equal(s2 URLSet) bool {
	return s1.Scheme == s2.Scheme &&
		s1.Port == s2.Port &&
		s1.Hosts.Equal(s2.Hosts)
}

func (us *URLSet) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteRune('"')
	if us != nil {
		urls := us.Hosts.List()
		sort.Strings(urls)
		for i, h := range urls {
			if i > 0 {
				b.WriteRune(',')
			}
			b.WriteString(us.Scheme)
			b.WriteString("://")
			b.WriteString(h)
			b.WriteString(":")
			b.WriteString(strconv.Itoa(us.Port))
		}
	}
	b.WriteRune('"')
	return []byte(b.String()), nil
}

func (us *URLSet) UnmarshalJSON(data []byte) error {
	if us == nil {
		return errors.New("jsontypes.URLSet: UnmarshalJSON on nil pointer")
	}

	n := len(data)
	if n < 2 {
		return fmt.Errorf("jsontypes.URLSet: UnmarshalJSON on invalid data %s", string(data))
	}
	if n == 2 && string(data) == `""` {
		return nil
	}
	us.Hosts = sets.NewString()

	urls := strings.Split(string(data[1:n-1]), ",")
	for _, rawurl := range urls {
		if u, err := url.Parse(rawurl); err == nil {
			us.Scheme = u.Scheme
			us.Hosts.Insert(u.Hostname())
			us.Port, err = strconv.Atoi(u.Port())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
