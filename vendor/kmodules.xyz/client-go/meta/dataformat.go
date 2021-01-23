package meta

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"sigs.k8s.io/yaml"
)

func DecodeObject(in map[string]interface{}, out interface{}) error {
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
