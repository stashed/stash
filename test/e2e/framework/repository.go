package framework

import (
	"math"
	"os"
	"strconv"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Framework) EventuallyRepository(kind string, objMeta metav1.ObjectMeta, replicas int) GomegaAsyncAssertion {
	return Eventually(func() []*api.Repository {
		return f.GetRepositories(kind, objMeta, replicas)
	})
}

func (f *Framework) GetRepositories(kind string, objMeta metav1.ObjectMeta, replicas int) []*api.Repository {
	repoNames := make([]string, 0)
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "minikube"
	}

	workload := api.LocalTypedReference{Name: objMeta.Name, Kind: kind}
	switch kind {
	case api.KindDeployment, api.KindReplicationController, api.KindReplicaSet, api.KindDaemonSet:
		repoNames = append(repoNames, workload.GetRepositoryCRDName("", nodeName))
	case api.KindStatefulSet:
		for i := 0; i < replicas; i++ {
			repoNames = append(repoNames, workload.GetRepositoryCRDName(objMeta.Name+"-"+strconv.Itoa(i), nodeName))
		}
	}
	repositories := make([]*api.Repository, 0)
	for _, repoName := range repoNames {
		obj, err := f.StashClient.StashV1alpha1().Repositories(objMeta.Namespace).Get(repoName, metav1.GetOptions{})
		if err == nil {
			repositories = append(repositories, obj)
		}
	}
	return repositories
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

func (f *Framework) BackupCountInRepositoriesStatus(repos []*api.Repository) int64 {
	var backupCount int64 = math.MaxInt64

	// use minimum backupCount among all repos
	for _, repo := range repos {
		if repo.Status.BackupCount < backupCount {
			backupCount = repo.Status.BackupCount
		}
	}
	return backupCount
}
