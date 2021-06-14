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
	"fmt"

	"github.com/mitchellh/mapstructure"
	"sigs.k8s.io/yaml"
)

func DecodeObject(in interface{}, out interface{}) error {
	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		TagName:  "json",
		Result:   out,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(in)
}

type DataFormat string

const (
	// Do not change format
	KeepFormat DataFormat = ""
	JsonFormat DataFormat = "json"
	YAMLFormat DataFormat = "yaml"
)

func NewDataFormat(format string, def DataFormat) DataFormat {
	switch format {
	case string(YAMLFormat):
		return YAMLFormat
	case string(JsonFormat):
		return JsonFormat
	default:
		return def
	}
}

func Marshal(v interface{}, format DataFormat) ([]byte, error) {
	if format == JsonFormat {
		return json.Marshal(v)
	} else if format == YAMLFormat {
		return yaml.Marshal(v)
	}
	return nil, fmt.Errorf("unknonw format: %v", format)
}
