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
