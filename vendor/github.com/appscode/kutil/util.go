package kutil

import (
	"github.com/hashicorp/go-version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"reflect"
	"time"
)

const (
	RetryInterval = 50 * time.Millisecond
	RetryTimeout  = 2 * time.Second
)

func IsPreferredAPIResource(c clientset.Interface, groupVersion, kind string) bool {
	if resourceList, err := c.Discovery().ServerPreferredResources(); err == nil {
		for _, resources := range resourceList {
			if resources.GroupVersion != groupVersion {
				continue
			}
			for _, resource := range resources.APIResources {
				if resources.GroupVersion == groupVersion && resource.Kind == kind {
					return true
				}
			}
		}
	}
	return false
}

func CheckAPIVersion(c clientset.Interface, constraint string) (bool, error) {
	info, err := c.Discovery().ServerVersion()
	if err != nil {
		return false, err
	}
	cond, err := version.NewConstraint(constraint)
	if err != nil {
		return false, err
	}
	v, err := version.NewVersion(info.GitVersion)
	if err != nil {
		return false, err
	}
	return cond.Check(v.ToMutator().ResetPrerelease().ResetMetadata().Done()), nil
}

func DeleteInBackground() *metav1.DeleteOptions {
	policy := metav1.DeletePropagationBackground
	return &metav1.DeleteOptions{PropagationPolicy: &policy}
}

func DeleteInForeground() *metav1.DeleteOptions {
	policy := metav1.DeletePropagationForeground
	return &metav1.DeleteOptions{PropagationPolicy: &policy}
}

func GetKind(v interface{}) string {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val.Type().Name()
}
