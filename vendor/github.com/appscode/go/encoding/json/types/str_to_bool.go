package types

import (
	"errors"
)

/*
StrToBool turns strings into bool when marshaled to Json. Empty strings are converted to false. Non-empty string, eg,
`"false"` will become True bool value. If already a json bool, then no change is made.

This can be used to turn a string to bool if you have existing Json data.
*/
type StrToBool bool

func (m *StrToBool) MarshalJSON() ([]byte, error) {
	a := *m
	if a {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}

func (m *StrToBool) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("jsontypes.StrToBool: UnmarshalJSON on nil pointer")
	}
	var err error
	if data[0] == '"' {
		// non empty string == true
		*m = (len(data) - 2) > 0
	} else {
		switch string(data) {
		case "true":
			*m = true
			err = nil
		case "false":
			*m = false
			err = nil
		default:
			err = errors.New("jsontypes.StrToBool: UnmarshalJSON failed for " + string(data))
		}
	}
	return err
}
