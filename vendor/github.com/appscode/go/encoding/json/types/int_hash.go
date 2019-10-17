package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/appscode/go/types"
)

/*
IntHash represents as int64 Generation and string Hash. It is json serialized into <int64>$<hash_string>.
*/
// +k8s:openapi-gen=true
type IntHash struct {
	generation int64
	hash       string
}

func ParseIntHash(v interface{}) (*IntHash, error) {
	switch m := v.(type) {
	case nil:
		return &IntHash{}, nil
	case int:
		return &IntHash{generation: int64(m)}, nil
	case int64:
		return &IntHash{generation: m}, nil
	case *int64:
		return &IntHash{generation: types.Int64(m)}, nil
	case IntHash:
		return &m, nil
	case *IntHash:
		return m, nil
	case string:
		return parseStringIntoIntHash(m)
	case *string:
		return parseStringIntoIntHash(types.String(m))
	default:
		return nil, fmt.Errorf("failed to parse type %s into IntHash", reflect.TypeOf(v).String())
	}
}

func parseStringIntoIntHash(s string) (*IntHash, error) {
	if s == "" {
		return &IntHash{}, nil
	}

	idx := strings.IndexRune(s, '$')
	switch {
	case idx <= 0:
		return nil, errors.New("missing generation")
	case idx == len(s)-1:
		return nil, errors.New("missing hash")
	default:
		i, err := strconv.ParseInt(s[:idx], 10, 64)
		if err != nil {
			return nil, err
		}
		h := s[idx+1:]
		return &IntHash{generation: i, hash: h}, nil
	}
}

func NewIntHash(i int64, h string) *IntHash { return &IntHash{generation: i, hash: h} }

func IntHashForGeneration(i int64) *IntHash { return &IntHash{generation: i} }

func IntHashForHash(h string) *IntHash { return &IntHash{hash: h} }

func (m IntHash) Generation() int64 {
	return m.generation
}

func (m IntHash) Hash() string {
	return m.hash
}

// IsZero returns true if the value is nil or time is zero.
func (m *IntHash) IsZero() bool {
	if m == nil {
		return true
	}
	return m.generation == 0 && m.hash == ""
}

func (m *IntHash) Equal(u *IntHash) bool {
	if m == nil {
		return u == nil
	}
	if u == nil { // t != nil
		return false
	}
	if m == u {
		return true
	}
	if m.generation == u.generation {
		return m.hash == u.hash
	}
	return false
}

func (m *IntHash) MatchGeneration(u *IntHash) bool {
	if m == nil {
		return u == nil
	}
	if u == nil { // t != nil
		return false
	}
	if m == u {
		return true
	}
	return m.generation == u.generation
}

func (m *IntHash) DeepCopyInto(out *IntHash) {
	*out = *m
}

func (m *IntHash) DeepCopy() *IntHash {
	if m == nil {
		return nil
	}
	out := new(IntHash)
	m.DeepCopyInto(out)
	return out
}

func (m IntHash) String() string {
	return fmt.Sprintf(`%d$%s`, m.generation, m.hash)
}

func (m *IntHash) MarshalJSON() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	if m.hash == "" {
		return json.Marshal(m.generation)
	}
	return json.Marshal(m.String())
}

func (m *IntHash) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("jsontypes.IntHash: UnmarshalJSON on nil pointer")
	}

	if data[0] == '"' {
		var s string
		err := json.Unmarshal(data, &s)
		if err != nil {
			return err
		}
		ih, err := ParseIntHash(s)
		if err != nil {
			return err
		}
		*m = *ih
		return nil
	} else if bytes.Equal(data, []byte("null")) {
		return nil
	}

	var i int64
	err := json.Unmarshal(data, &i)
	if err != nil {
		return err
	}
	m.generation = i
	return nil
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (_ IntHash) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (_ IntHash) OpenAPISchemaFormat() string { return "" }

// MarshalQueryParameter converts to a URL query parameter value
func (m IntHash) MarshalQueryParameter() (string, error) {
	if m.IsZero() {
		// Encode unset/nil objects as an empty string
		return "", nil
	}
	return m.String(), nil
}
