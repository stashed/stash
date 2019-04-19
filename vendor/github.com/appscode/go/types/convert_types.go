package types

import "time"

// StringP returns a pointer to the string value passed in.
func StringP(v string) *string {
	return &v
}

// String returns the value of the string pointer passed in or
// "" if the pointer is nil.
func String(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

// StringPSlice converts a slice of string values into a slice of
// string pointers
func StringPSlice(src []string) []*string {
	dst := make([]*string, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// StringSlice converts a slice of string pointers into a slice of
// string values
func StringSlice(src []*string) []string {
	dst := make([]string, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// StringPMap converts a string map of string values into a string
// map of string pointers
func StringPMap(src map[string]string) map[string]*string {
	dst := make(map[string]*string)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// StringMap converts a string map of string pointers into a string
// map of string values
func StringMap(src map[string]*string) map[string]string {
	dst := make(map[string]string)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

var trueP = BoolP(true)

// FalseP returns a pointer to `true` boolean value.
func TrueP() *bool {
	return trueP
}

var falseP = BoolP(false)

// FalseP returns a pointer to `false` boolean value.
func FalseP() *bool {
	return falseP
}

// BoolP returns a pointer to the bool value passed in.
func BoolP(v bool) *bool {
	return &v
}

// Bool returns the value of the bool pointer passed in or
// false if the pointer is nil.
func Bool(v *bool) bool {
	if v != nil {
		return *v
	}
	return false
}

// BoolPSlice converts a slice of bool values into a slice of
// bool pointers
func BoolPSlice(src []bool) []*bool {
	dst := make([]*bool, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// BoolSlice converts a slice of bool pointers into a slice of
// bool values
func BoolSlice(src []*bool) []bool {
	dst := make([]bool, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// BoolPMap converts a string map of bool values into a string
// map of bool pointers
func BoolPMap(src map[string]bool) map[string]*bool {
	dst := make(map[string]*bool)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// BoolMap converts a string map of bool pointers into a string
// map of bool values
func BoolMap(src map[string]*bool) map[string]bool {
	dst := make(map[string]bool)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// IntP returns a pointer to the int value passed in.
func IntP(v int) *int {
	return &v
}

// Int returns the value of the int pointer passed in or
// 0 if the pointer is nil.
func Int(v *int) int {
	if v != nil {
		return *v
	}
	return 0
}

// IntPSlice converts a slice of int values into a slice of
// int pointers
func IntPSlice(src []int) []*int {
	dst := make([]*int, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// IntSlice converts a slice of int pointers into a slice of
// int values
func IntSlice(src []*int) []int {
	dst := make([]int, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// IntPMap converts a string map of int values into a string
// map of int pointers
func IntPMap(src map[string]int) map[string]*int {
	dst := make(map[string]*int)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// IntMap converts a string map of int pointers into a string
// map of int values
func IntMap(src map[string]*int) map[string]int {
	dst := make(map[string]int)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// UIntP returns a pointer to the uint value passed in.
func UIntP(v uint) *uint {
	return &v
}

// UInt returns the value of the uint pointer passed in or
// 0 if the pointer is nil.
func UInt(v *uint) uint {
	if v != nil {
		return *v
	}
	return 0
}

// UIntPSlice converts a slice of uint values into a slice of
// uint pointers
func UIntPSlice(src []uint) []*uint {
	dst := make([]*uint, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// UIntSlice converts a slice of uint pointers into a slice of
// uint values
func UIntSlice(src []*uint) []uint {
	dst := make([]uint, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// UIntPMap converts a string map of uint values into a string
// map of uint pointers
func UIntPMap(src map[string]uint) map[string]*uint {
	dst := make(map[string]*uint)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// UIntMap converts a string map of uint pointers into a string
// map of uint values
func UIntMap(src map[string]*uint) map[string]uint {
	dst := make(map[string]uint)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// Int32P returns a pointer to the int32 value passed in.
func Int32P(v int32) *int32 {
	return &v
}

// Int32 returns the value of the int32 pointer passed in or
// 0 if the pointer is nil.
func Int32(v *int32) int32 {
	if v != nil {
		return *v
	}
	return 0
}

// Int32PSlice converts a slice of int32 values into a slice of
// int32 pointers
func Int32PSlice(src []int32) []*int32 {
	dst := make([]*int32, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Int32Slice converts a slice of int32 pointers into a slice of
// int32 values
func Int32Slice(src []*int32) []int32 {
	dst := make([]int32, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// Int32PMap converts a string map of int32 values into a string
// map of int32 pointers
func Int32PMap(src map[string]int32) map[string]*int32 {
	dst := make(map[string]*int32)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// Int32Map converts a string map of int32 pointers into a string
// map of int32 values
func Int32Map(src map[string]*int32) map[string]int32 {
	dst := make(map[string]int32)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// Int64P returns a pointer to the int64 value passed in.
func Int64P(v int64) *int64 {
	return &v
}

// Int64 returns the value of the int64 pointer passed in or
// 0 if the pointer is nil.
func Int64(v *int64) int64 {
	if v != nil {
		return *v
	}
	return 0
}

// Int64PSlice converts a slice of int64 values into a slice of
// int64 pointers
func Int64PSlice(src []int64) []*int64 {
	dst := make([]*int64, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Int64Slice converts a slice of int64 pointers into a slice of
// int64 values
func Int64Slice(src []*int64) []int64 {
	dst := make([]int64, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// Int64PMap converts a string map of int64 values into a string
// map of int64 pointers
func Int64PMap(src map[string]int64) map[string]*int64 {
	dst := make(map[string]*int64)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// Int64Map converts a string map of int64 pointers into a string
// map of int64 values
func Int64Map(src map[string]*int64) map[string]int64 {
	dst := make(map[string]int64)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// Float64P returns a pointer to the float64 value passed in.
func Float64P(v float64) *float64 {
	return &v
}

// Float64 returns the value of the float64 pointer passed in or
// 0 if the pointer is nil.
func Float64(v *float64) float64 {
	if v != nil {
		return *v
	}
	return 0
}

// Float64PSlice converts a slice of float64 values into a slice of
// float64 pointers
func Float64PSlice(src []float64) []*float64 {
	dst := make([]*float64, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Float64Slice converts a slice of float64 pointers into a slice of
// float64 values
func Float64Slice(src []*float64) []float64 {
	dst := make([]float64, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// Float64PMap converts a string map of float64 values into a string
// map of float64 pointers
func Float64PMap(src map[string]float64) map[string]*float64 {
	dst := make(map[string]*float64)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// Float64Map converts a string map of float64 pointers into a string
// map of float64 values
func Float64Map(src map[string]*float64) map[string]float64 {
	dst := make(map[string]float64)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// TimeP returns a pointer to the time.Time value passed in.
func TimeP(v time.Time) *time.Time {
	return &v
}

// Time returns the value of the time.Time pointer passed in or
// time.Time{} if the pointer is nil.
func Time(v *time.Time) time.Time {
	if v != nil {
		return *v
	}
	return time.Time{}
}

// TimeUnixMilli returns a Unix timestamp in milliseconds from "January 1, 1970 UTC".
// The result is undefined if the Unix time cannot be represented by an int64.
// Which includes calling TimeUnixMilli on a zero Time is undefined.
//
// This utility is useful for service API's such as CloudWatch Logs which require
// their unix time values to be in milliseconds.
//
// See Go stdlib https://golang.org/pkg/time/#Time.UnixNano for more information.
func TimeUnixMilli(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond/time.Nanosecond)
}

// TimePSlice converts a slice of time.Time values into a slice of
// time.Time pointers
func TimePSlice(src []time.Time) []*time.Time {
	dst := make([]*time.Time, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// TimeSlice converts a slice of time.Time pointers into a slice of
// time.Time values
func TimeSlice(src []*time.Time) []time.Time {
	dst := make([]time.Time, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// TimePMap converts a string map of time.Time values into a string
// map of time.Time pointers
func TimePMap(src map[string]time.Time) map[string]*time.Time {
	dst := make(map[string]*time.Time)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// TimeMap converts a string map of time.Time pointers into a string
// map of time.Time values
func TimeMap(src map[string]*time.Time) map[string]time.Time {
	dst := make(map[string]time.Time)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}
