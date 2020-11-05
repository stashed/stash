package types

import (
	"errors"
	"strconv"
)

type BoolYo bool

func (m *BoolYo) MarshalJSON() ([]byte, error) {
	a := *m
	if a {
		return []byte(`"true"`), nil
	}
	return []byte(`"false"`), nil
}

func (m *BoolYo) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("jsontypes.BoolYo: UnmarshalJSON on nil pointer")
	}

	n := len(data)
	var in string
	if data[0] == '"' && data[n-1] == '"' {
		in = string(data[1 : n-1])
	} else {
		in = string(data)
	}
	v, err := strconv.ParseBool(in)
	*m = BoolYo(v)
	return err
}
