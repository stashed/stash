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
	"unicode/utf8"

	"gomodules.xyz/encoding/json"
)

/*
StrYo turns non-strings into into a string by adding quotes around it into bool,
when marshaled to Json. If input is already string, no change is done.
*/
type StrYo string

func (m *StrYo) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("jsontypes.StrYo: UnmarshalJSON on nil pointer")
	}

	if data[0] == '"' {
		var s string
		err := json.Unmarshal(data, &s)
		if err != nil {
			return err
		}
		*m = StrYo(s)
		return nil
	} else if data[0] == '{' {
		return errors.New("jsontypes.StrYo: Expected string, found object")
	} else if data[0] == '[' {
		return errors.New("jsontypes.StrYo: Expected string, found array")
	} else if bytes.Equal(data, []byte("null")) {
		*m = ""
		return nil
	}
	d := string(data)
	if utf8.ValidString(d) {
		*m = StrYo(d)
		return nil
	}
	return errors.New("jsontypes.StrYo: Found invalid utf8 byte array")
}
