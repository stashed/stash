package types

import (
	"bytes"
	"encoding/json"
	"errors"
)

/*
        GO => Json
        [] => `[]`
     ["a"] => `"a"`
["a", "b"] => `["a","b"]`
*/
type ArrayOrString []string

func (m *ArrayOrString) MarshalJSON() ([]byte, error) {
	a := *m
	n := len(a)
	var buf bytes.Buffer
	if n == 1 {
		buf.WriteString(`"`)
		buf.WriteString(a[0])
		buf.WriteString(`"`)
	} else {
		buf.WriteString(`[`)

		for i := 0; i < n; i++ {
			if i > 0 {
				buf.WriteString(`,`)
			}
			buf.WriteString(`"`)
			buf.WriteString(a[i])
			buf.WriteString(`"`)
		}

		buf.WriteString(`]`)
	}
	return buf.Bytes(), nil
}

func (m *ArrayOrString) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("jsontypes.ArrayOrString: UnmarshalJSON on nil pointer")
	}
	var err error
	if data[0] == '[' {
		var a []string
		err = json.Unmarshal(data, &a)
		if err == nil {
			*m = a
		}
	} else {
		*m = append((*m)[0:0], string(data[1:len(data)-1]))
	}
	return err
}
