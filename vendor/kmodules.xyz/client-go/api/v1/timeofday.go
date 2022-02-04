/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"encoding/json"
	"time"

	fuzz "github.com/google/gofuzz"
)

// TimeOfDay is a wrapper around time.Time which supports correct
// marshaling to YAML and JSON.  Wrappers are provided for many
// of the factory methods that the time package offers.
//
// +protobuf.options.marshal=false
// +protobuf.as=Timestamp
// +protobuf.options.(gogoproto.goproto_stringer)=false
type TimeOfDay struct {
	time.Time `protobuf:"-"`
}

// DeepCopyInto creates a deep-copy of the TimeOfDay value.  The underlying time.Time
// type is effectively immutable in the time API, so it is safe to
// copy-by-assign, despite the presence of (unexported) Pointer fields.
func (t *TimeOfDay) DeepCopyInto(out *TimeOfDay) {
	*out = *t
}

// NewTime returns a wrapped instance of the provided time
func NewTime(t time.Time) TimeOfDay {
	utc := t.UTC()
	return TimeOfDay{time.Date(0, 0, 0, utc.Hour(), utc.Minute(), utc.Second(), 0, time.UTC)}
}

// Date returns the TimeOfDay corresponding to the supplied parameters
// by wrapping time.Date.
func Date(hour, min, sec int) TimeOfDay {
	return TimeOfDay{time.Date(0, 0, 0, hour, min, sec, 0, time.UTC)}
}

// Now returns the current local time.
func Now() TimeOfDay {
	utc := time.Now().UTC()
	return TimeOfDay{time.Date(0, 0, 0, utc.Hour(), utc.Minute(), utc.Second(), 0, time.UTC)}
}

// IsZero returns true if the value is nil or time is zero.
func (t *TimeOfDay) IsZero() bool {
	if t == nil {
		return true
	}
	return t.Time.IsZero()
}

// Before reports whether the time instant t is before u.
func (t *TimeOfDay) Before(u *TimeOfDay) bool {
	if t != nil && u != nil {
		return t.Time.Before(u.Time)
	}
	return false
}

// Equal reports whether the time instant t is equal to u.
func (t *TimeOfDay) Equal(u *TimeOfDay) bool {
	if t == nil && u == nil {
		return true
	}
	if t != nil && u != nil {
		return t.Time.Equal(u.Time)
	}
	return false
}

// Unix returns the local time corresponding to the given Unix time
// by wrapping time.Unix.
func Unix(sec int64, nsec int64) TimeOfDay {
	return TimeOfDay{time.Unix(sec, nsec)}
}

// Copy returns a copy of the TimeOfDay at second-level precision.
func (t TimeOfDay) Copy() TimeOfDay {
	copied, _ := time.Parse(time.Kitchen, t.Format(time.Kitchen))
	return TimeOfDay{copied}
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (t *TimeOfDay) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		t.Time = time.Time{}
		return nil
	}

	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	pt, err := time.Parse(time.Kitchen, str)
	if err != nil {
		return err
	}

	t.Time = pt.Local()
	return nil
}

// UnmarshalQueryParameter converts from a URL query parameter value to an object
func (t *TimeOfDay) UnmarshalQueryParameter(str string) error {
	if len(str) == 0 {
		t.Time = time.Time{}
		return nil
	}
	// Tolerate requests from older clients that used JSON serialization to build query params
	if len(str) == 4 && str == "null" {
		t.Time = time.Time{}
		return nil
	}

	pt, err := time.Parse(time.Kitchen, str)
	if err != nil {
		return err
	}

	t.Time = pt.Local()
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (t TimeOfDay) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		// Encode unset/nil objects as JSON's "null".
		return []byte("null"), nil
	}
	buf := make([]byte, 0, len(time.Kitchen)+2)
	buf = append(buf, '"')
	// time cannot contain non escapable JSON characters
	buf = t.UTC().AppendFormat(buf, time.Kitchen)
	buf = append(buf, '"')
	return buf, nil
}

// ToUnstructured implements the value.UnstructuredConverter interface.
func (t TimeOfDay) ToUnstructured() interface{} {
	if t.IsZero() {
		return nil
	}
	buf := make([]byte, 0, len(time.Kitchen))
	buf = t.UTC().AppendFormat(buf, time.Kitchen)
	return string(buf)
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (_ TimeOfDay) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (_ TimeOfDay) OpenAPISchemaFormat() string { return "time" }

// MarshalQueryParameter converts to a URL query parameter value
func (t TimeOfDay) MarshalQueryParameter() (string, error) {
	if t.IsZero() {
		// Encode unset/nil objects as an empty string
		return "", nil
	}

	return t.UTC().Format(time.Kitchen), nil
}

// Fuzz satisfies fuzz.Interface.
func (t *TimeOfDay) Fuzz(c fuzz.Continue) {
	if t == nil {
		return
	}
	// Allow for about 1000 years of randomness.  Leave off nanoseconds
	// because JSON doesn't represent them so they can't round-trip
	// properly.
	t.Time = time.Unix(c.Rand.Int63n(1000*365*24*60*60), 0)
}

// ensure Time implements fuzz.Interface
var _ fuzz.Interface = &TimeOfDay{}
