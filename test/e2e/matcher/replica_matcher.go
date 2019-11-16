/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package matcher

import (
	"fmt"

	"github.com/onsi/gomega/types"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
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
	case *apps.ReplicaSet:
		return *obj.Spec.Replicas == int32(matcher.expected), nil
	case *apps.Deployment:
		return *obj.Spec.Replicas == int32(matcher.expected), nil
	//case *extensions.DaemonSet:
	//	return matcher.find(obj.Spec.Template.Spec.Containers)
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
	return fmt.Sprintf("Expected\n\t%#v\n not to have\n\t%#v replica", actual, matcher.expected)
}
