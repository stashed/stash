package matcher

import (
	"fmt"

	"github.com/onsi/gomega/types"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
)

func HaveReplica(expected int) types.GomegaMatcher {
	return &replicaMatcher{
		expected: expected,
	}
}

type replicaMatcher struct {
	expected int
}

func (matcher *replicaMatcher) Match(actual interface{}) (success bool, err error) {
	switch obj := actual.(type) {
	case *core.ReplicationController:
		return *obj.Spec.Replicas == int32(matcher.expected), nil
	case *extensions.ReplicaSet:
		return *obj.Spec.Replicas == int32(matcher.expected), nil
	case *extensions.Deployment:
		return *obj.Spec.Replicas == int32(matcher.expected), nil
	//case *extensions.DaemonSet:
	//	return matcher.find(obj.Spec.Template.Spec.Containers)
	case *apps.Deployment:
		return *obj.Spec.Replicas == int32(matcher.expected), nil
	//case *apps.StatefulSet:
	//	return matcher.find(obj.Spec.Template.Spec.Containers)

	default:
		return false, fmt.Errorf("Unknown object type")
	}
}

func (matcher *replicaMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\n to have \n\t%#v replica", actual, matcher.expected)
}

func (matcher *replicaMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to have\n\t%#v replica", actual, matcher.expected)
}
