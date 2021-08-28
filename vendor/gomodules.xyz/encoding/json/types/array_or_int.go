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
	"strconv"

	"gomodules.xyz/encoding/json"
)

/*
    GO => Json
    [] => `[]`
   [1] => `1`
[1, 2] => `[1,2]`
*/
type ArrayOrInt []int

func (m *ArrayOrInt) MarshalJSON() ([]byte, error) {
	a := *m
	n := len(a)
	var buf bytes.Buffer
	if n == 1 {
		buf.WriteString(strconv.Itoa(a[0]))
	} else {
		buf.WriteString(`[`)

		for i := 0; i < n; i++ {
			if i > 0 {
				buf.WriteString(`,`)
			}
			buf.WriteString(strconv.Itoa(a[i]))
		}

		buf.WriteString(`]`)
	}
	return buf.Bytes(), nil
}

func (m *ArrayOrInt) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("jsontypes.ArrayOrInt: UnmarshalJSON on nil pointer")
	}
	var err error
	if data[0] == '[' {
		var a []int
		err = json.Unmarshal(data, &a)
		if err == nil {
			*m = a
		}
	} else {
		v, _ := strconv.Atoi(string(data))
		*m = append((*m)[0:0], v)
	}
	return err
}
