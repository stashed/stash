package framework

import (
	"math"
	"os"
	"strconv"

	api "github.com/appscode/stash/apis/stash/v1alpha1"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KindMetaReplicas struct {
	Kind     string
	Meta     metav1.ObjectMeta
	Replicas int
}

func (f *Framework) EventuallyRepository(kmr KindMetaReplicas) GomegaAsyncAssertion {
	return Eventually(func() []*api.Repository {
		return f.GetRepositories(kmr)
	})
}

func (f *Framework) GetRepositories(kmr KindMetaReplicas) []*api.Repository {
	repoNames := make([]string, 0)
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "minikube"
	}

	workload := api.LocalTypedReference{Name: kmr.Meta.Name, Kind: kmr.Kind}
	switch kmr.Kind {
	case api.KindDeployment, api.KindReplicationController, api.KindReplicaSet, api.KindDaemonSet:
		repoNames = append(repoNames, workload.GetRepositoryCRDName("", nodeName))
	case api.KindStatefulSet:
		for i := 0; i < kmr.Replicas; i++ {
			repoNames = append(repoNames, workload.GetRepositoryCRDName(kmr.Meta.Name+"-"+strconv.Itoa(i), nodeName))
		}
	}
	repositories := make([]*api.Repository, 0)
	for _, repoName := range repoNames {
		obj, err := f.StashClient.StashV1alpha1().Repositories(kmr.Meta.Namespace).Get(repoName, metav1.GetOptions{})
		if err == nil {
			repositories = append(repositories, obj)
		}
	}
	return repositories
}

func (f *Framework) DeleteRepositories(kmrs []KindMetaReplicas) {
	repositories := make([]*api.Repository, 0)
	for _, kmr := range kmrs {
		repos := f.GetRepositories(kmr)
		repositories = append(repositories, repos...)
	}
	for _, repo := range repositories {
		err:=f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Delete(repo.Name, deleteInForeground())
		Expect(err).NotTo(HaveOccurred())
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
