package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"unicode/utf8"
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
