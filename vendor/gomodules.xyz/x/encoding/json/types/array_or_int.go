package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
)

/*
    GO => Json
    [] => `[]`
   [1] => `1`
[1, 2] => `[1,2]`
*/
type ArrayOrInt []int

func (m *ArrayOrInt) MarshalJSON() ([]byte, error) {
	a := *m
	n := len(a)
	var buf bytes.Buffer
	if n == 1 {
		buf.WriteString(strconv.Itoa(a[0]))
	} else {
		buf.WriteString(`[`)

		for i := 0; i < n; i++ {
			if i > 0 {
				buf.WriteString(`,`)
			}
			buf.WriteString(strconv.Itoa(a[i]))
		}

		buf.WriteString(`]`)
	}
	return buf.Bytes(), nil
}

func (m *ArrayOrInt) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("jsontypes.ArrayOrInt: UnmarshalJSON on nil pointer")
	}
	var err error
	if data[0] == '[' {
		var a []int
		err = json.Unmarshal(data, &a)
		if err == nil {
			*m = a
		}
	} else {
		v, _ := strconv.Atoi(string(data))
		*m = append((*m)[0:0], v)
	}
	return err
}
