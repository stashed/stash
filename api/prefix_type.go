package api

import (
	"encoding/json"
	"errors"
)

type PrefixType int

const (
	Smart PrefixType = iota
	NodeName
	PodName
	None
)

func (m PrefixType) String() string {
	switch m {
	case Smart:
		return "Smart"
	case NodeName:
		return "NodeName"
	case PodName:
		return "PodName"
	case None:
		return "None"
	}
	return "<invalid>"
}

func (m PrefixType) MarshalJSON() ([]byte, error) {
	switch m {
	case Smart:
		return []byte(`"Smart"`), nil
	case NodeName:
		return []byte(`"NodeName"`), nil
	case PodName:
		return []byte(`"PodName"`), nil
	case None:
		return []byte(`"None"`), nil
	}
	return nil, errors.New("api.PrefixType: Invalid PrefixType")
}

func (m *PrefixType) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("api.PrefixType: UnmarshalJSON on nil pointer")
	}
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	switch s {
	case "Smart", "":
		*m = Smart
	case "NodeName":
		*m = NodeName
	case "PodName":
		*m = PodName
	case "None":
		*m = None
	default:
		return errors.New("api.PrefixType: Invalid PrefixType")
	}
	return nil
}
