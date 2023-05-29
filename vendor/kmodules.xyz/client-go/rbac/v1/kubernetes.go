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

package v1

import (
	jsoniter "github.com/json-iterator/go"
	rbac "k8s.io/api/rbac/v1"
)

var json = jsoniter.ConfigFastest

func UpsertSubjects(subjects []rbac.Subject, upsert ...rbac.Subject) []rbac.Subject {
	for i := range upsert {
		var found bool
		for j := range subjects {
			if subjects[j] == upsert[i] {
				found = true
				break
			}
		}
		if !found {
			subjects = append(subjects, upsert[i])
		}
	}
	return subjects
}

func RemoveSubjects(subjects []rbac.Subject, rm ...rbac.Subject) []rbac.Subject {
	var out []rbac.Subject
	for _, x := range rm {
		out = subjects[:0]
		for _, y := range subjects {
			if y != x {
				out = append(out, y)
			}
		}
		subjects = out
	}
	return out
}
