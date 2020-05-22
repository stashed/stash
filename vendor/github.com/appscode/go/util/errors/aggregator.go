package errors

import (
	"fmt"
)

// Forked from: https://github.com/kubernetes/apimachinery/blob/master/pkg/util/errors/errors.go

// Aggregate represents an object that contains multiple errors, but does not
// necessarily have singular semantic meaning.
type Aggregate interface {
	error
	Errors() []error
}

// NewAggregate converts a slice of errors into an Aggregate interface, which
// is itself an implementation of the error interface.  If the slice is empty,
// this returns nil.
// It will check if any of the element of input error list is nil, to avoid
// nil pointer panic when call Error().
func NewAggregate(errlist []error) Aggregate {
	if len(errlist) == 0 {
		return nil
	}
	// In case of input error list contains nil
	var errs []error
	for _, e := range errlist {
		if e != nil {
			errs = append(errs, e)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return aggregate(errs)
}

// This helper implements the error and Errors interfaces.  Keeping it private
// prevents people from making an aggregate of 0 errors, which is not
// an error, but does satisfy the error interface.
type aggregate []error

// Error is part of the error interface.
func (agg aggregate) Error() string {
	if len(agg) == 0 {
		// This should never happen, really.
		return ""
	}
	if len(agg) == 1 {
		return agg[0].Error()
	}
	result := fmt.Sprintf("[%s", agg[0].Error())
	for i := 1; i < len(agg); i++ {
		result += fmt.Sprintf(", %s", agg[i].Error())
	}
	result += "]"
	return result
}

// Errors is part of the Aggregate interface.
func (agg aggregate) Errors() []error {
	return []error(agg)
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}
