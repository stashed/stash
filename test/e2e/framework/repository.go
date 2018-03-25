package framework

import (
	"os"
	"strconv"
	"strings"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyRepository(kind string, objMeta metav1.ObjectMeta, replicas int) GomegaAsyncAssertion {
	return Eventually(func() []*api.Repository {
		repoNames := make([]string, 0)
		switch kind {
		case api.KindDeployment, api.KindReplicationController, api.KindReplicaSet:
			repoNames = append(repoNames, strings.ToLower(kind)+"."+objMeta.Name)
		case api.KindStatefulSet:
			for i := 0; i < replicas; i++ {
				repoNames = append(repoNames, strings.ToLower(kind)+"."+objMeta.Name+"-"+strconv.Itoa(i))
			}
		case api.KindDaemonSet:
			nodeName := os.Getenv("NODE_NAME")
			if nodeName == "" {
				nodeName = "minikube"
			}
			repoNames = append(repoNames, strings.ToLower(kind)+"."+objMeta.Name+"."+nodeName)
		}
		repositories := make([]*api.Repository, 0)
		for _, repoName := range repoNames {
			obj, _ := f.StashClient.StashV1alpha1().Repositories(objMeta.Namespace).Get(repoName, metav1.GetOptions{})

			repositories = append(repositories, obj)
		}
		return repositories
	})
}

func (f *Framework) DeleteRepositories() {
	repoList, err := f.StashClient.StashV1alpha1().Repositories(f.namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Infof(err.Error())
		return
	}
	for _, repo := range repoList.Items {
		f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Delete(repo.Name, deleteInBackground())
	}
}
