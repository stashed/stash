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
	core "k8s.io/api/core/v1"
)

func HaveEvent(expected string) types.GomegaMatcher {
	return &eventMatcher{
		expected: expected,
	}
}

type eventMatcher struct {
	expected string
}

func (matcher *eventMatcher) Match(actual interface{}) (success bool, err error) {
	events := actual.([]core.Event)
	return matcher.find(events)
}

func (matcher *eventMatcher) find(events []core.Event) (success bool, err error) {
	for _, e := range events {
		if e.Reason == matcher.expected {
			return true, nil
		}
	}
	return false, nil
}

func (matcher *eventMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\n to contain event \n\t%#v", actual, matcher.expected)
}

func (matcher *eventMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\n not to contain the event\n\t%#v", actual, matcher.expected)
}
