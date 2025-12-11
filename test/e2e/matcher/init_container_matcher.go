/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

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

func HaveInitContainer(expected string) types.GomegaMatcher {
	return &initContainerMatcher{
		expected: expected,
	}
}

type initContainerMatcher struct {
	expected string
}

func (matcher *initContainerMatcher) Match(actual any) (success bool, err error) {
	switch obj := actual.(type) {
	case *core.Pod:
		return matcher.find(obj.Spec.InitContainers)
	case *apps.Deployment:
		return matcher.find(obj.Spec.Template.Spec.InitContainers)
	case *apps.DaemonSet:
		return matcher.find(obj.Spec.Template.Spec.InitContainers)
	case *apps.StatefulSet:
		return matcher.find(obj.Spec.Template.Spec.InitContainers)
	case []core.Container:
		return matcher.find(obj)
	default:
		return false, fmt.Errorf("unknown object type")
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

func (matcher *initContainerMatcher) FailureMessage(actual any) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\n to contain sidecar container \n\t%#v", actual, matcher.expected)
}

func (matcher *initContainerMatcher) NegatedFailureMessage(actual any) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\n not to contain the sidecar container\n\t%#v", actual, matcher.expected)
}
