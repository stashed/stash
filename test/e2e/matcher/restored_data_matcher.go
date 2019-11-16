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

	"github.com/appscode/go/sets"
	"github.com/onsi/gomega/types"
)

func BeSameAs(sample sets.String) types.GomegaMatcher {
	return &recoveredDataMatcher{
		sample: sample,
	}
}

type recoveredDataMatcher struct {
	sample sets.String
}

func (matcher *recoveredDataMatcher) Match(actual interface{}) (success bool, err error) {
	recoveredData := actual.(sets.String)
	for data := range recoveredData {
		if !matcher.sample.Has(data) {
			return false, nil
		}
	}
	return true, nil
}

func (matcher *recoveredDataMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\tRecovered data: %v\n to  be same as Sample data:  %v\n\t", actual, matcher.sample)
}

func (matcher *recoveredDataMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\tRecovered data: %v\n not to be same as Sample data:  %v\n\t", actual, matcher.sample)
}
