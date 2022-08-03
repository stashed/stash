package sprig

import (
	"bytes"
	"fmt"

	"gomodules.xyz/encoding/json"
	"gomodules.xyz/jsonpath"
)

func jsonpathFn(expr string, data interface{}, jsonoutput ...bool) (interface{}, error) {
	enableJSONoutput := len(jsonoutput) > 0 && jsonoutput[0]

	jp := jsonpath.New("jp")
	if err := jp.Parse(expr); err != nil {
		return nil, fmt.Errorf("unrecognized column definition %q", expr)
	}
	jp.AllowMissingKeys(true)
	jp.EnableJSONOutput(enableJSONoutput)

	var buf bytes.Buffer
	err := jp.Execute(&buf, data)
	if err != nil {
		return nil, err
	}

	if enableJSONoutput {
		var v []interface{}
		err = json.Unmarshal(buf.Bytes(), &v)
		return v, err
	}
	return buf.String(), err
}
