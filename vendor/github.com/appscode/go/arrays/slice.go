package arrays

import (
	"reflect"
)

func Reverse(slice interface{}) ([]interface{}, error) {
	sliceA, err := InterfaceToSlice(slice)
	if err != nil {
		return nil, err
	}
	for i := len(sliceA)/2 - 1; i >= 0; i-- {
		opp := len(sliceA) - 1 - i
		sliceA[i], sliceA[opp] = sliceA[opp], sliceA[i]
	}
	return sliceA, nil
}

func Filter(slice interface{}, f func(interface{}) bool) ([]interface{}, error) {
	sliceA, err := InterfaceToSlice(slice)
	if err != nil {
		return nil, err
	}
	b := sliceA[:0]
	for _, x := range sliceA {
		if f(x) {
			b = append(b, x)
		}
	}
	return b, nil
}

func Contains(slice interface{}, value interface{}) (bool, int) {
	s, err := InterfaceToSlice(slice)
	if err != nil {
		return false, -1
	}

	for i, v := range s {
		if reflect.DeepEqual(v, value) {
			return true, i
		}
	}
	return false, -1
}
