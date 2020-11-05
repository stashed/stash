package pointer

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

// UintP returns a pointer to the uint value passed in.
func UintP(v uint) *uint {
	return &v
}

// Uint returns the value of the uint pointer passed in or
// 0 if the pointer is nil.
func Uint(v *uint) uint {
	if v != nil {
		return *v
	}
	return 0
}

// UintPSlice converts a slice of uint values uinto a slice of
// uint pointers
func UintPSlice(src []uint) []*uint {
	dst := make([]*uint, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// UintSlice converts a slice of uint pointers uinto a slice of
// uint values
func UintSlice(src []*uint) []uint {
	dst := make([]uint, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// UintPMap converts a string map of uint values uinto a string
// map of uint pointers
func UintPMap(src map[string]uint) map[string]*uint {
	dst := make(map[string]*uint)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// UintMap converts a string map of uint pointers uinto a string
// map of uint values
func UintMap(src map[string]*uint) map[string]uint {
	dst := make(map[string]uint)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// Int8P returns a pointer to the int8 value passed in.
func Int8P(v int8) *int8 {
	return &v
}

// Int8 returns the value of the int8 pointer passed in or
// 0 if the pointer is nil.
func Int8(v *int8) int8 {
	if v != nil {
		return *v
	}
	return 0
}

// Int8PSlice converts a slice of int8 values into a slice of
// int8 pointers
func Int8PSlice(src []int8) []*int8 {
	dst := make([]*int8, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Int8Slice converts a slice of int8 pointers into a slice of
// int8 values
func Int8Slice(src []*int8) []int8 {
	dst := make([]int8, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// Int8PMap converts a string map of int8 values into a string
// map of int8 pointers
func Int8PMap(src map[string]int8) map[string]*int8 {
	dst := make(map[string]*int8)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// Int8Map converts a string map of int8 pointers into a string
// map of int8 values
func Int8Map(src map[string]*int8) map[string]int8 {
	dst := make(map[string]int8)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// Int16P returns a pointer to the int16 value passed in.
func Int16P(v int16) *int16 {
	return &v
}

// Int16 returns the value of the int16 pointer passed in or
// 0 if the pointer is nil.
func Int16(v *int16) int16 {
	if v != nil {
		return *v
	}
	return 0
}

// Int16PSlice converts a slice of int16 values into a slice of
// int16 pointers
func Int16PSlice(src []int16) []*int16 {
	dst := make([]*int16, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Int16Slice converts a slice of int16 pointers into a slice of
// int16 values
func Int16Slice(src []*int16) []int16 {
	dst := make([]int16, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// Int16PMap converts a string map of int16 values into a string
// map of int16 pointers
func Int16PMap(src map[string]int16) map[string]*int16 {
	dst := make(map[string]*int16)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// Int16Map converts a string map of int16 pointers into a string
// map of int16 values
func Int16Map(src map[string]*int16) map[string]int16 {
	dst := make(map[string]int16)
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

// Uint8P returns a pointer to the uint8 value passed in.
func Uint8P(v uint8) *uint8 {
	return &v
}

// Uint8 returns the value of the uint8 pointer passed in or
// 0 if the pointer is nil.
func Uint8(v *uint8) uint8 {
	if v != nil {
		return *v
	}
	return 0
}

// Uint8PSlice converts a slice of uint8 values into a slice of
// uint8 pointers
func Uint8PSlice(src []uint8) []*uint8 {
	dst := make([]*uint8, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Uint8Slice converts a slice of uint8 pointers into a slice of
// uint8 values
func Uint8Slice(src []*uint8) []uint8 {
	dst := make([]uint8, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// Uint8PMap converts a string map of uint8 values into a string
// map of uint8 pointers
func Uint8PMap(src map[string]uint8) map[string]*uint8 {
	dst := make(map[string]*uint8)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// Uint8Map converts a string map of uint8 pointers into a string
// map of uint8 values
func Uint8Map(src map[string]*uint8) map[string]uint8 {
	dst := make(map[string]uint8)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// Uint16P returns a pointer to the uint16 value passed in.
func Uint16P(v uint16) *uint16 {
	return &v
}

// Uint16 returns the value of the uint16 pointer passed in or
// 0 if the pointer is nil.
func Uint16(v *uint16) uint16 {
	if v != nil {
		return *v
	}
	return 0
}

// Uint16PSlice converts a slice of uint16 values into a slice of
// uint16 pointers
func Uint16PSlice(src []uint16) []*uint16 {
	dst := make([]*uint16, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Uint16Slice converts a slice of uint16 pointers into a slice of
// uint16 values
func Uint16Slice(src []*uint16) []uint16 {
	dst := make([]uint16, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// Uint16PMap converts a string map of uint16 values into a string
// map of uint16 pointers
func Uint16PMap(src map[string]uint16) map[string]*uint16 {
	dst := make(map[string]*uint16)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// Uint16Map converts a string map of uint16 pointers into a string
// map of uint16 values
func Uint16Map(src map[string]*uint16) map[string]uint16 {
	dst := make(map[string]uint16)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// Uint32P returns a pointer to the uint32 value passed in.
func Uint32P(v uint32) *uint32 {
	return &v
}

// Uint32 returns the value of the uint32 pointer passed in or
// 0 if the pointer is nil.
func Uint32(v *uint32) uint32 {
	if v != nil {
		return *v
	}
	return 0
}

// Uint32PSlice converts a slice of uint32 values into a slice of
// uint32 pointers
func Uint32PSlice(src []uint32) []*uint32 {
	dst := make([]*uint32, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Uint32Slice converts a slice of uint32 pointers into a slice of
// uint32 values
func Uint32Slice(src []*uint32) []uint32 {
	dst := make([]uint32, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// Uint32PMap converts a string map of uint32 values into a string
// map of uint32 pointers
func Uint32PMap(src map[string]uint32) map[string]*uint32 {
	dst := make(map[string]*uint32)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// Uint32Map converts a string map of uint32 pointers into a string
// map of uint32 values
func Uint32Map(src map[string]*uint32) map[string]uint32 {
	dst := make(map[string]uint32)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// Uint64P returns a pointer to the uint64 value passed in.
func Uint64P(v uint64) *uint64 {
	return &v
}

// Uint64 returns the value of the uint64 pointer passed in or
// 0 if the pointer is nil.
func Uint64(v *uint64) uint64 {
	if v != nil {
		return *v
	}
	return 0
}

// Uint64PSlice converts a slice of uint64 values into a slice of
// uint64 pointers
func Uint64PSlice(src []uint64) []*uint64 {
	dst := make([]*uint64, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Uint64Slice converts a slice of uint64 pointers into a slice of
// uint64 values
func Uint64Slice(src []*uint64) []uint64 {
	dst := make([]uint64, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// Uint64PMap converts a string map of uint64 values into a string
// map of uint64 pointers
func Uint64PMap(src map[string]uint64) map[string]*uint64 {
	dst := make(map[string]*uint64)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// Uint64Map converts a string map of uint64 pointers into a string
// map of uint64 values
func Uint64Map(src map[string]*uint64) map[string]uint64 {
	dst := make(map[string]uint64)
	for k, val := range src {
		if val != nil {
			dst[k] = *val
		}
	}
	return dst
}

// Float32P returns a pointer to the float32 value passed in.
func Float32P(v float32) *float32 {
	return &v
}

// Float32 returns the value of the float32 pointer passed in or
// 0 if the pointer is nil.
func Float32(v *float32) float32 {
	if v != nil {
		return *v
	}
	return 0
}

// Float32PSlice converts a slice of float32 values into a slice of
// float32 pointers
func Float32PSlice(src []float32) []*float32 {
	dst := make([]*float32, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Float32Slice converts a slice of float32 pointers into a slice of
// float32 values
func Float32Slice(src []*float32) []float32 {
	dst := make([]float32, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != nil {
			dst[i] = *(src[i])
		}
	}
	return dst
}

// Float32PMap converts a string map of float32 values into a string
// map of float32 pointers
func Float32PMap(src map[string]float32) map[string]*float32 {
	dst := make(map[string]*float32)
	for k, val := range src {
		v := val
		dst[k] = &v
	}
	return dst
}

// Float32Map converts a string map of float32 pointers into a string
// map of float32 values
func Float32Map(src map[string]*float32) map[string]float32 {
	dst := make(map[string]float32)
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

// SecondsTime converts an int64 pointer to a time.Time value
// representing seconds since Epoch or time.Time{} if the pointer is nil.
func SecondsTime(v *int64) time.Time {
	if v != nil {
		return time.Unix((*v / 1000), 0)
	}
	return time.Time{}
}

// MillisecondsTime converts an int64 pointer to a time.Time value
// representing milliseconds sinch Epoch or time.Time{} if the pointer is nil.
func MillisecondsTime(v *int64) time.Time {
	if v != nil {
		return time.Unix(0, (*v * 1000000))
	}
	return time.Time{}
}

// TimeUnixMilli returns a Unix timestamp in milliseconds from "January 1, 1970 UTC".
// The result is undefined if the Unix time cannot be represented by an int64.
// Which includes calling TimeUnixMilli on a zero TimeP is undefined.
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
