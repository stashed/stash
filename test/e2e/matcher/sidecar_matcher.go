package matcher

import (
	"fmt"

	"github.com/onsi/gomega/types"
	apps "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
)

func HaveSidecar(expected string) types.GomegaMatcher {
	return &sidecarMatcher{
		expected: expected,
	}
}

type sidecarMatcher struct {
	expected string
}

func (matcher *sidecarMatcher) Match(actual interface{}) (success bool, err error) {
	switch obj := actual.(type) {
	case *apiv1.Pod:
		return matcher.find(obj.Spec.Containers)
	case *apiv1.ReplicationController:
		return matcher.find(obj.Spec.Template.Spec.Containers)
	case *extensions.ReplicaSet:
		return matcher.find(obj.Spec.Template.Spec.Containers)
	case *extensions.Deployment:
		return matcher.find(obj.Spec.Template.Spec.Containers)
	case *extensions.DaemonSet:
		return matcher.find(obj.Spec.Template.Spec.Containers)
	case *apps.Deployment:
		return matcher.find(obj.Spec.Template.Spec.Containers)
	case *apps.StatefulSet:
		return matcher.find(obj.Spec.Template.Spec.Containers)
	case []apiv1.Container:
		return matcher.find(obj)
	default:
		return false, fmt.Errorf("Unknown object type")
	}
}

func (matcher *sidecarMatcher) find(containers []apiv1.Container) (success bool, err error) {
	for _, c := range containers {
		if c.Name == matcher.expected {
			return true, nil
		}
	}
	return false, nil
}

func (matcher *sidecarMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto contain sidecar container \n\t%#v", actual, matcher.expected)
}

func (matcher *sidecarMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to contain the sidecar container\n\t%#v", actual, matcher.expected)
}
