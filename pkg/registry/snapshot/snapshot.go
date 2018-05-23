package snapshot

import (
	"github.com/appscode/stash/apis/repositories"
	stash "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/client/clientset/versioned"
	"github.com/appscode/stash/pkg/util"
	"github.com/pkg/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"
	restconfig "k8s.io/client-go/rest"
)

type REST struct {
	stashClient versioned.Interface
	kubeClient  kubernetes.Interface
	config      *restconfig.Config
}

var _ rest.Getter = &REST{}
var _ rest.Lister = &REST{}
var _ rest.GracefulDeleter = &REST{}

func NewREST(config *restconfig.Config) *REST {
	return &REST{
		stashClient: versioned.NewForConfigOrDie(config),
		kubeClient:  kubernetes.NewForConfigOrDie(config),
		config:      config,
	}
}

func (r *REST) New() runtime.Object {
	return &repositories.Snapshot{}
}

func (r *REST) Get(ctx apirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, errors.New("missing namespace")
	}
	if len(name) < 9 {
		return nil, errors.New("invalid snapshot name")
	}

	repoName, snapshotId, err := util.GetRepoNameAndSnapshotID(name)
	if err != nil {
		return nil, err
	}

	repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(repoName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.New("respective repository not found. error:" + err.Error())
	}

	snapshots := make([]repositories.Snapshot, 0)
	if repo.Spec.Backend.Local != nil {
		snapshots, err = r.getSnapshotsFromSidecar(repo, []string{snapshotId})
	} else {
		snapshots, err = r.GetSnapshots(repo, []string{snapshotId})
	}
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return nil, errors.New("no resource found")
	}

	snapshot := &repositories.Snapshot{}
	snapshot = &snapshots[0]
	return snapshot, nil
}

func (r *REST) NewList() runtime.Object {
	return &repositories.SnapshotList{}
}

func (r *REST) List(ctx apirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, errors.New("missing namespace")
	}

	repos, err := r.stashClient.StashV1alpha1().Repositories(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var selectedRepos []stash.Repository
	if options.LabelSelector != nil {
		for _, r := range repos.Items {
			repoLabels := make(map[string]string)
			repoLabels = r.Labels
			repoLabels["repository"] = r.Name
			if options.LabelSelector.Matches(labels.Set(repoLabels)) {
				selectedRepos = append(selectedRepos, r)
			}
		}
	} else {
		selectedRepos = repos.Items
	}

	snapshotList := &repositories.SnapshotList{
		Items: make([]repositories.Snapshot, 0),
	}
	for _, repo := range selectedRepos {
		var snapshots []repositories.Snapshot
		if repo.Spec.Backend.Local != nil {
			snapshots, err = r.getSnapshotsFromSidecar(&repo, nil)
			if err != nil {
				return nil, err
			}
		} else {
			snapshots, err = r.GetSnapshots(&repo, nil)
			if err != nil {
				return nil, err
			}
		}
		snapshotList.Items = append(snapshotList.Items, snapshots...)
	}

	// k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructured_list.go
	// unstructured.UnstructuredList{}
	return snapshotList, nil
}

func (r *REST) Delete(ctx apirequest.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, false, errors.New("missing namespace")
	}
	repoName, snapshotId, err := util.GetRepoNameAndSnapshotID(name)
	if err != nil {
		return nil, false, err
	}
	repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(repoName, metav1.GetOptions{})
	if err != nil {
		return nil, false, errors.New("respective repository not found. error:" + err.Error())
	}

	if repo.Spec.Backend.Local != nil {
		err = r.forgetSnapshotsFromSidecar(repo, []string{snapshotId})
	} else {
		err = r.ForgetSnapshots(repo, []string{snapshotId})
	}
	if err != nil {
		return nil, false, err
	}

	return nil, true, nil
}
