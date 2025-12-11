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
	"gomodules.xyz/x/strings"
)

func BeSameAs(sample []string) types.GomegaMatcher {
	return &recoveredDataMatcher{
		sample: sample,
	}
}

type recoveredDataMatcher struct {
	sample []string
}

func (matcher *recoveredDataMatcher) Match(actual any) (success bool, err error) {
	recoveredData := actual.([]string)
	for _, data := range recoveredData {
		if !strings.Contains(matcher.sample, data) {
			return false, nil
		}
	}
	return true, nil
}

func (matcher *recoveredDataMatcher) FailureMessage(actual any) (message string) {
	return fmt.Sprintf("Expected\n\tRecovered data: %v\n to  be same as Sample data:  %v\n\t", actual, matcher.sample)
}

func (matcher *recoveredDataMatcher) NegatedFailureMessage(actual any) (message string) {
	return fmt.Sprintf("Expected\n\tRecovered data: %v\n not to be same as Sample data:  %v\n\t", actual, matcher.sample)
}
