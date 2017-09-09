package version

import (
	"bytes"
	"encoding/json"
)

// UnmarshalJSON implements the json.Unmarshaller interface.
func (v *Version) UnmarshalJSON(value []byte) error {
	var str string
	err := json.Unmarshal(value, &str)
	if err != nil {
		return err
	}
	vj, err := NewVersion(str)
	if err != nil {
		return err
	}
	*v = *vj
	return nil
}

// MarshalJSON implements the json.Marshaller interface.
func (v Version) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	e.SetEscapeHTML(false)
	err := e.Encode(v.String())
	// https://stackoverflow.com/a/36320146/244009
	return bytes.TrimSpace(buf.Bytes()), err
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (c *Constraints) UnmarshalJSON(value []byte) error {
	var str string

	err := json.Unmarshal(value, &str)
	if err != nil {
		return err
	}
	cj, err := NewConstraint(str)
	if err != nil {
		return err
	}
	*c = cj
	return nil
}

// MarshalJSON implements the json.Marshaller interface.
func (c Constraints) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	e.SetEscapeHTML(false)
	err := e.Encode(c.String())
	return bytes.TrimSpace(buf.Bytes()), err
}
