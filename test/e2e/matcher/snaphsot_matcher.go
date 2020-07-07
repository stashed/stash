/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package matcher

import (
	"fmt"
	"strings"

	snap_api "stash.appscode.dev/apimachinery/apis/repositories/v1alpha1"

	"github.com/onsi/gomega/types"
)

func HavePrefixInName(prefix string) types.GomegaMatcher {
	return &namePrefixMatcher{
		prefix: prefix,
	}
}

type namePrefixMatcher struct {
	prefix string
}

func (matcher *namePrefixMatcher) Match(actual interface{}) (success bool, err error) {
	snapshotList := actual.(*snap_api.SnapshotList)
	for _, snap := range snapshotList.Items {
		if !strings.HasPrefix(snap.Name, matcher.prefix) {
			return false, nil
		}
	}
	return true, nil
}

func (matcher *namePrefixMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\tSnapshots name\n to  have prefix %v\n\t", matcher.prefix)
}

func (matcher *namePrefixMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\tSnapshots name\n not to have prefix %v\n\t", &matcher)
}
