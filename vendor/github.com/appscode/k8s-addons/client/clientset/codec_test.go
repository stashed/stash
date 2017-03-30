package clientset

import (
	"fmt"
	"reflect"
	"testing"

	aci "github.com/appscode/k8s-addons/api"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	"k8s.io/kubernetes/pkg/runtime"
)

func TestDefaultGroupVersion(t *testing.T) {
	i := &aci.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}

	gv, err := unversioned.ParseGroupVersion("appscode.com/v1beta1")
	if err != nil {
		fmt.Println(err)
	}
	// if appscode.com/v1beta1 is not enabled, return an error
	if !registered.IsEnabledVersion(gv) {
		fmt.Println("appscode.com/v1beta1 is not enabled")
	}

	fmt.Println(*i)
}

func TestSetDefault(t *testing.T) {
	metadata := &unversioned.TypeMeta{
		Kind:       "Ingress",
		APIVersion: "appscode.com/v1beta1",
	}
	var obj runtime.Object

	obj, err := setDefaultType(metadata)
	fmt.Println(obj, err)
	assert.NotNil(t, obj)
	fmt.Println(reflect.ValueOf(obj).Type().String())
}
