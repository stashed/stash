package snapshot

import (
	"strings"

	api "github.com/appscode/stash/apis/repositories/v1alpha1"
	"github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/client/clientset/versioned"
	"github.com/pkg/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	snapshotIDs := make([]string, 0)
	snapshotIDs = append(snapshotIDs, name[len(name)-8:])

	repoName := strings.TrimSuffix(name, name[len(name)-9:])

	repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(repoName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.New("respective repository not found. error:" + err.Error())
	}

	snapshots := make([]api.Snapshot, 0)
	if repo.Spec.Backend.Local != nil {
		snapshots, err = r.getSnapshotsFromSidecar(repo, snapshotIDs)
	} else {
		snapshots, err = r.GetSnapshots(repo, snapshotIDs)
	}
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return nil, errors.New("no resource found")
	}

	snapshotList := &api.SnapshotList{}
	snapshotList.Items = snapshots
	return snapshotList, nil
}

func (r *REST) NewList() runtime.Object {
	return &api.SnapshotList{}
}

func (r *REST) List(ctx apirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {

	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, errors.New("missing namespace")
	}

	labelSelector := options.LabelSelector.String()
	repos := make([]v1alpha1.Repository, 0)

	// if the request is for snapshots of specific repositories then use Get method to get those repositories
	// otherwise use labelSelector to list all repositories that match the selectors.
	if strings.Contains(labelSelector, "repository=") {
		repoNames := make([]string, 0)
		labels := strings.Split(labelSelector, ",")
		for _, label := range labels {
			if strings.Contains(label, "repository=") {
				repoNames = append(repoNames, strings.TrimPrefix(label, "repository="))
			}
		}

		for _, reponame := range repoNames {
			repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(reponame, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			repos = append(repos, *repo)
		}
	} else {
		repoList, err := r.stashClient.StashV1alpha1().Repositories(ns).List(metav1.ListOptions{LabelSelector: options.LabelSelector.String()})
		if err != nil {
			return nil, err
		}
		repos = repoList.Items
	}

	snapshotList := &api.SnapshotList{}
	snapshots := make([]api.Snapshot, 0)
	snapshotIDs := make([]string, 0)
	var err error
	for _, repo := range repos {
		if repo.Spec.Backend.Local != nil {
			snapshots, err = r.getSnapshotsFromSidecar(&repo, snapshotIDs)
			if err != nil {
				return nil, err
			}
		} else {
			snapshots, err = r.GetSnapshots(&repo, snapshotIDs)
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
	if len(name) < 9 {
		return nil, errors.New("invalid snapshot name")
	}
	snapshotIDs := make([]string, 0)
	snapshotIDs = append(snapshotIDs, name[len(name)-8:])

	repoName := strings.TrimSuffix(name, name[len(name)-9:])

	repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(repoName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.New("respective repository not found. error:" + err.Error())
	}

	if repo.Spec.Backend.Local != nil {
		err = r.forgetSnapshotsFromSidecar(repo, snapshotIDs)
	} else {
		err = r.ForgetSnapshots(repo, snapshotIDs)
	}
	if err != nil {
		return nil, err
	}

	return nil, nil
}
