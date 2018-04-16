package yaml

import (
	"bytes"
	"encoding/json"
	"unicode"

	"github.com/ghodss/yaml"
)

// ToJSON converts a single YAML document into a JSON document
// or returns an error. If the document appears to be JSON the
// YAML decoding path is not used (so that error messages are)
// JSON specific.
func ToJSON(data []byte) ([]byte, error) {
	if hasJSONPrefix(data) {
		return data, nil
	}
	b, err := yaml.YAMLToJSON(data)
	if err != nil {
		return b, err
	}
	return b, nil

}

var jsonPrefix = []byte("{")

// hasJSONPrefix returns true if the provided buffer appears to start with
// a JSON open brace.
func hasJSONPrefix(buf []byte) bool {
	trim := bytes.TrimLeftFunc(buf, unicode.IsSpace)
	return bytes.HasPrefix(trim, jsonPrefix)
}

func Marshal(o interface{}) ([]byte, error) {
	data, err := yaml.Marshal(o)
	if err != nil {
		return nil, err
	}

	allFields := make(map[string]interface{})
	err = yaml.Unmarshal(data, &allFields)
	if err != nil {
		return nil, err
	}
	nonDefaultFields := make(map[string]interface{})
	for k, v := range allFields {
		switch u := v.(type) {
		case int, int32, int64:
			if u != 0 {
				nonDefaultFields[k] = u
			}
		case string:
			if u != "" {
				nonDefaultFields[k] = u
			}
		case bool:
			if u != false {
				nonDefaultFields[k] = u
			}
		default:
			nonDefaultFields[k] = v
		}
	}

	data, err = yaml.Marshal(nonDefaultFields)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func Unmarshal(b []byte, i interface{}) error {
	return yaml.Unmarshal(b, i)
}

func ToString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	res, err := yaml.JSONToYAML(b)
	if err != nil {
		return ""
	}
	return string(res)
}
