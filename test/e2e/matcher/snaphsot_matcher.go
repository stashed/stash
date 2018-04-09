package matcher

import (
	"fmt"
	"strings"

	strings2 "github.com/appscode/go/strings"
	snap_api "github.com/appscode/stash/apis/repositories/v1alpha1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	"github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CameFromAllTestRepositories(stashClient cs.Interface, namespace string) types.GomegaMatcher {
	return &repositoryMatcher{
		stashClient: stashClient,
		namespace:   namespace,
	}
}

type repositoryMatcher struct {
	stashClient cs.Interface
	namespace   string
}

func (matcher *repositoryMatcher) Match(actual interface{}) (success bool, err error) {
	snapshotList := actual.(*snap_api.SnapshotList)
	snapshotRepoNames := make([]string, 0)
	for _, snap := range snapshotList.Items {
		snapshotRepoNames = append(snapshotRepoNames, strings.TrimSuffix(snap.Name, snap.Name[len(snap.Name)-9:]))
	}

	repoList, err := matcher.stashClient.StashV1alpha1().Repositories(matcher.namespace).List(metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, repo := range repoList.Items {
		if !strings2.Contains(snapshotRepoNames, repo.Name) {
			return false, nil
		}
	}
	return true, nil
}

func (matcher *repositoryMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\tSnapshots\nto came from all test repositories \n\t")
}

func (matcher *repositoryMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\tSnapshots\nnot to came from all test repositories\n\t")
}

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
	return fmt.Sprintf("Expected\n\tSnapshots name\nto  have prefix %v\n\t", matcher.prefix)
}

func (matcher *namePrefixMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\tSnapshots name\nnot to have prefix %v\n\t", &matcher)
}
