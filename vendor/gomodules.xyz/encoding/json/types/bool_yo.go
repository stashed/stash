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
	"errors"
	"strconv"
)

type BoolYo bool

func (m *BoolYo) MarshalJSON() ([]byte, error) {
	a := *m
	if a {
		return []byte(`"true"`), nil
	}
	return []byte(`"false"`), nil
}

func (m *BoolYo) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("jsontypes.BoolYo: UnmarshalJSON on nil pointer")
	}

	n := len(data)
	var in string
	if data[0] == '"' && data[n-1] == '"' {
		in = string(data[1 : n-1])
	} else {
		in = string(data)
	}
	v, err := strconv.ParseBool(in)
	*m = BoolYo(v)
	return err
}
