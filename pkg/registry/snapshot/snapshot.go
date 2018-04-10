package snapshot

import (
	api "github.com/appscode/stash/apis/repositories/v1alpha1"
	"github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/client/clientset/versioned"
	"github.com/pkg/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
var _ rest.Deleter = &REST{}
var _ rest.GroupVersionKindProvider = &REST{}

func NewREST(config *restconfig.Config) *REST {
	return &REST{
		stashClient: versioned.NewForConfigOrDie(config),
		kubeClient:  kubernetes.NewForConfigOrDie(config),
		config:      config,
	}
}

func (r *REST) New() runtime.Object {
	return &api.Snapshot{}
}

func (r *REST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return api.SchemeGroupVersion.WithKind(api.ResourceKindSnapshot)
}

func (r *REST) Get(ctx apirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, errors.New("missing namespace")
	}
	if len(name) < 9 {
		return nil, errors.New("invalid snapshot name")
	}

	repoName, snapshotId, err := GetRepoNameAndSnapshotID(name)
	if err != nil {
		return nil, err
	}

	repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(repoName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.New("respective repository not found. error:" + err.Error())
	}

	snapshots := make([]api.Snapshot, 0)
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

	snapshot := &api.Snapshot{}
	snapshot = &snapshots[0]
	return snapshot, nil
}

func (r *REST) NewList() runtime.Object {
	return &api.SnapshotList{}
}

func (r *REST) List(ctx apirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {

	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, errors.New("missing namespace")
	}

	repositories, err := r.stashClient.StashV1alpha1().Repositories(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var selectedRepos []v1alpha1.Repository
	if options.LabelSelector != nil {
		for _, r := range repositories.Items {
			repoLabels := make(map[string]string)
			repoLabels = r.Labels
			repoLabels["repository"] = r.Name
			if options.LabelSelector.Matches(labels.Set(repoLabels)) {
				selectedRepos = append(selectedRepos, r)
			}
		}
	} else {
		selectedRepos = repositories.Items
	}

	snapshotList := &api.SnapshotList{}
	snapshots := make([]api.Snapshot, 0)
	for _, repo := range selectedRepos {
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
	if len(snapshotList.Items) == 0 {
		return nil, errors.New("no resource found")
	}
	return snapshotList, nil
}

func (r *REST) Delete(ctx apirequest.Context, name string) (runtime.Object, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, errors.New("missing namespace")
	}
	repoName, snapshotId, err := GetRepoNameAndSnapshotID(name)
	if err != nil {
		return nil, err
	}
	repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(repoName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.New("respective repository not found. error:" + err.Error())
	}

	if repo.Spec.Backend.Local != nil {
		err = r.forgetSnapshotsFromSidecar(repo, []string{snapshotId})
	} else {
		err = r.ForgetSnapshots(repo, []string{snapshotId})
	}
	if err != nil {
		return nil, err
	}

	return nil, nil
}
