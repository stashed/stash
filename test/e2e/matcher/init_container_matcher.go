package matcher

import (
	"fmt"

	"github.com/onsi/gomega/types"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

func HaveInitContainer(expected string) types.GomegaMatcher {
	return &initContainerMatcher{
		expected: expected,
	}
}

type initContainerMatcher struct {
	expected string
}

func (matcher *initContainerMatcher) Match(actual interface{}) (success bool, err error) {
	switch obj := actual.(type) {
	case *core.Pod:
		return matcher.find(obj.Spec.InitContainers)
	case *core.ReplicationController:
		return matcher.find(obj.Spec.Template.Spec.InitContainers)
	case *apps.ReplicaSet:
		return matcher.find(obj.Spec.Template.Spec.InitContainers)
	case *apps.Deployment:
		return matcher.find(obj.Spec.Template.Spec.InitContainers)
	case *apps.DaemonSet:
		return matcher.find(obj.Spec.Template.Spec.InitContainers)
	case *apps.StatefulSet:
		return matcher.find(obj.Spec.Template.Spec.InitContainers)
	case []core.Container:
		return matcher.find(obj)
	default:
		return false, fmt.Errorf("Unknown object type")
	}
}

func (matcher *initContainerMatcher) find(containers []core.Container) (success bool, err error) {
	for _, c := range containers {
		if c.Name == matcher.expected {
			return true, nil
		}
	}
	return false, nil
}

func (matcher *initContainerMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\n to contain sidecar container \n\t%#v", actual, matcher.expected)
}

func (matcher *initContainerMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\n not to contain the sidecar container\n\t%#v", actual, matcher.expected)
}
