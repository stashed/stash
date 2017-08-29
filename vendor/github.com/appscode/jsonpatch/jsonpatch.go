package jsonpatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

var errBadJSONDoc = fmt.Errorf("Invalid JSON Document")

type JsonPatchOperation struct {
	Operation string      `json:"op"`
	Path      string      `json:"path"`
	Value     interface{} `json:"value,omitempty"`
}

func (j *JsonPatchOperation) Json() string {
	b, _ := json.Marshal(j)
	return string(b)
}

func (j *JsonPatchOperation) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("{")
	b.WriteString(fmt.Sprintf(`"op":"%s"`, j.Operation))
	b.WriteString(fmt.Sprintf(`,"path":"%s"`, j.Path))
	// Consider omitting Value for non-nullable operations.
	if j.Value != nil || j.Operation == "replace" || j.Operation == "add" {
		v, err := json.Marshal(j.Value)
		if err != nil {
			return nil, err
		}
		b.WriteString(`,"value":`)
		b.Write(v)
	}
	b.WriteString("}")
	return b.Bytes(), nil
}

type ByPath []JsonPatchOperation

func (a ByPath) Len() int           { return len(a) }
func (a ByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPath) Less(i, j int) bool { return a[i].Path < a[j].Path }

func NewPatch(operation, path string, value interface{}) JsonPatchOperation {
	return JsonPatchOperation{Operation: operation, Path: path, Value: value}
}

// CreatePatch creates a patch as specified in http://jsonpatch.com/
//
// 'a' is original, 'b' is the modified document. Both are to be given as json encoded content.
// The function will return an array of JsonPatchOperations
//
// An error will be returned if any of the two documents are invalid.
func CreatePatch(a, b []byte) ([]JsonPatchOperation, error) {
	aI := map[string]interface{}{}
	bI := map[string]interface{}{}
	err := json.Unmarshal(a, &aI)
	if err != nil {
		return nil, errBadJSONDoc
	}
	err = json.Unmarshal(b, &bI)
	if err != nil {
		return nil, errBadJSONDoc
	}
	return diff(aI, bI, "", []JsonPatchOperation{})
}

// Returns true if the values matches (must be json types)
// The types of the values must match, otherwise it will always return false
// If two map[string]interface{} are given, all elements must match.
func matchesValue(av, bv interface{}) bool {
	if reflect.TypeOf(av) != reflect.TypeOf(bv) {
		return false
	}
	switch at := av.(type) {
	case string:
		bt := bv.(string)
		if bt == at {
			return true
		}
	case float64:
		bt := bv.(float64)
		if bt == at {
			return true
		}
	case bool:
		bt := bv.(bool)
		if bt == at {
			return true
		}
	case map[string]interface{}:
		bt := bv.(map[string]interface{})
		for key := range at {
			if !matchesValue(at[key], bt[key]) {
				return false
			}
		}
		for key := range bt {
			if !matchesValue(at[key], bt[key]) {
				return false
			}
		}
		return true
	case []interface{}:
		bt := bv.([]interface{})
		if len(bt) != len(at) {
			return false
		}
		for key := range at {
			if !matchesValue(at[key], bt[key]) {
				return false
			}
		}
		for key := range bt {
			if !matchesValue(at[key], bt[key]) {
				return false
			}
		}
		return true
	}
	return false
}

func makePath(path string, newPart interface{}) string {
	if path == "" {
		return fmt.Sprintf("/%v", newPart)
	} else {
		if strings.HasSuffix(path, "/") {
			path = path + fmt.Sprintf("%v", newPart)
		} else {
			path = path + fmt.Sprintf("/%v", newPart)
		}
	}
	return path
}

// diff returns the (recursive) difference between a and b as an array of JsonPatchOperations.
func diff(a, b map[string]interface{}, path string, patch []JsonPatchOperation) ([]JsonPatchOperation, error) {
	for key, bv := range b {
		p := makePath(path, key)
		av, ok := a[key]
		// value was added
		if !ok {
			patch = append(patch, NewPatch("add", p, bv))
			continue
		}
		// If types have changed, replace completely
		if reflect.TypeOf(av) != reflect.TypeOf(bv) {
			patch = append(patch, NewPatch("replace", p, bv))
			continue
		}
		// Types are the same, compare values
		var err error
		patch, err = handleValues(av, bv, p, patch)
		if err != nil {
			return nil, err
		}
	}
	// Now add all deleted values as nil
	for key := range a {
		_, found := b[key]
		if !found {
			p := makePath(path, key)

			patch = append(patch, NewPatch("remove", p, nil))
		}
	}
	return patch, nil
}

func handleValues(av, bv interface{}, p string, patch []JsonPatchOperation) ([]JsonPatchOperation, error) {
	var err error
	switch at := av.(type) {
	case map[string]interface{}:
		bt := bv.(map[string]interface{})
		patch, err = diff(at, bt, p, patch)
		if err != nil {
			return nil, err
		}
	case string, float64, bool:
		if !matchesValue(av, bv) {
			patch = append(patch, NewPatch("replace", p, bv))
		}
	case []interface{}:
		bt := bv.([]interface{})
		if isSimpleArray(at) && isSimpleArray(bt) {
			patch = append(patch, compareEditDistance(at, bt, p)...)
		} else {
			n := min(len(at), len(bt))
			for i := len(at) - 1; i >= n; i-- {
				patch = append(patch, NewPatch("remove", makePath(p, i), nil))
			}
			for i := n; i < len(bt); i++ {
				patch = append(patch, NewPatch("add", makePath(p, i), bt[i]))
			}
			for i := 0; i < n; i++ {
				var err error
				patch, err = handleValues(at[i], bt[i], makePath(p, i), patch)
				if err != nil {
					return nil, err
				}
			}
		}
	case nil:
		switch bv.(type) {
		case nil:
			// Both nil, fine.
		default:
			patch = append(patch, NewPatch("add", p, bv))
		}
	default:
		panic(fmt.Sprintf("Unknown type:%T ", av))
	}
	return patch, nil
}

func isBasicType(a interface{}) bool {
	switch a.(type) {
	case string, float64, bool:
	default:
		return false
	}
	return true
}

func isSimpleArray(a []interface{}) bool {
	for i := range a {
		switch a[i].(type) {
		case string, float64, bool:
		default:
			val := reflect.ValueOf(a[i])
			if val.Kind() == reflect.Map {
				for _, k := range val.MapKeys() {
					av := val.MapIndex(k)
					if av.Kind() == reflect.Ptr || av.Kind() == reflect.Interface {
						if av.IsNil() {
							continue
						}
						av = av.Elem()
					}
					if av.Kind() != reflect.String && av.Kind() != reflect.Float64 && av.Kind() != reflect.Bool {
						return false
					}
				}
				return true
			}
			return false
		}
	}
	return true
}

// https://en.wikipedia.org/wiki/Wagner%E2%80%93Fischer_algorithm
// Adapted from https://github.com/texttheater/golang-levenshtein
func compareEditDistance(s, t []interface{}, p string) []JsonPatchOperation {
	m := len(s)
	n := len(t)

	d := make([][]int, m+1)
	for i := 0; i <= m; i++ {
		d[i] = make([]int, n+1)
		d[i][0] = i
	}
	for j := 0; j <= n; j++ {
		d[0][j] = j
	}

	for j := 1; j <= n; j++ {
		for i := 1; i <= m; i++ {
			if reflect.DeepEqual(s[i-1], t[j-1]) {
				d[i][j] = d[i-1][j-1] // no op required
			} else {
				del := d[i-1][j] + 1
				add := d[i][j-1] + 1
				rep := d[i-1][j-1] + 1
				d[i][j] = min(rep, min(add, del))
			}
		}
	}

	return backtrace(s, t, p, m, n, d)
}

func min(x int, y int) int {
	if y < x {
		return y
	}
	return x
}

func backtrace(s, t []interface{}, p string, i int, j int, matrix [][]int) []JsonPatchOperation {
	if i > 0 && matrix[i-1][j]+1 == matrix[i][j] {
		return append(backtrace(s, t, p, i-1, j, matrix), NewPatch("remove", makePath(p, i-1), nil))
	}
	if j > 0 && matrix[i][j-1]+1 == matrix[i][j] {
		return append(backtrace(s, t, p, i, j-1, matrix), NewPatch("add", makePath(p, i), t[j-1]))
	}
	if i > 0 && j > 0 && matrix[i-1][j-1]+1 == matrix[i][j] {
		if isBasicType(s[0]) {
			return append(backtrace(s, t, p, i-1, j-1, matrix), NewPatch("replace", makePath(p, i-1), t[j-1]))
		}

		p2, _ := handleValues(s[j-1], t[j-1], makePath(p, i-1), []JsonPatchOperation{})
		return append(backtrace(s, t, p, i-1, j-1, matrix), p2...)
	}
	if i > 0 && j > 0 && matrix[i-1][j-1] == matrix[i][j] {
		return backtrace(s, t, p, i-1, j-1, matrix)
	}
	return []JsonPatchOperation{}
}
