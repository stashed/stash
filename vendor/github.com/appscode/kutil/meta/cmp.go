package meta

import (
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	cmpOptions = []cmp.Option{
		cmp.Comparer(func(x, y resource.Quantity) bool {
			return x.Cmp(y) == 0
		}),
		cmp.Comparer(func(x, y *metav1.Time) bool {
			if x == nil && y == nil {
				return true
			}
			if x != nil && y != nil {
				return x.Time.Equal(y.Time)
			}
			return false
		}),
	}
)

func Diff(x, y interface{}) string {
	return cmp.Diff(x, y, cmpOptions...)
}

func Equal(x, y interface{}) bool {
	return cmp.Equal(x, y, cmpOptions...)
}
