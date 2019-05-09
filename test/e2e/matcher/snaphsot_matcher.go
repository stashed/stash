package matcher

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega/types"
	snap_api "stash.appscode.dev/stash/apis/repositories/v1alpha1"
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
