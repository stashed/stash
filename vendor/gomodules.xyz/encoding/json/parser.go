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

package json

import (
	"strings"

	"github.com/fatih/structs"
)

func ExtractTag(s interface{}, field string, jsonTag ...string) (tag string, inline, exists bool) {
	f, ok := structs.New(s).FieldOk(field)
	if !ok {
		return "", false, false
	}
	tagName := "json"
	if len(jsonTag) > 0 {
		tagName = jsonTag[0]
	}
	return ParseTag(f.Tag(tagName))
}

func ParseTag(in string) (tag string, inline, exists bool) {
	exists = in != ""
	parts := strings.Split(in, ",")
	tag = parts[0]
	for _, opt := range parts[1:] {
		if opt == "inline" {
			inline = true
			break
		}
	}
	return
}
