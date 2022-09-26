/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package snapshot

import (
	"context"

	"stash.appscode.dev/apimachinery/apis/repositories"
	repov1alpha1 "stash.appscode.dev/apimachinery/apis/repositories/v1alpha1"
	stash "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/util"

	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"
	restconfig "k8s.io/client-go/rest"
	meta_util "kmodules.xyz/client-go/meta"
)

const (
	KeyRepository = "repository"
	KeyHostname   = "hostname"
)

type REST struct {
	stashClient versioned.Interface
	kubeClient  kubernetes.Interface
	config      *restconfig.Config
	convertor   rest.TableConvertor
}

var (
	_ rest.Scoper                   = &REST{}
	_ rest.Storage                  = &REST{}
	_ rest.Getter                   = &REST{}
	_ rest.Lister                   = &REST{}
	_ rest.GracefulDeleter          = &REST{}
	_ rest.GroupVersionKindProvider = &REST{}
	_ rest.CategoriesProvider       = &REST{}
)

func NewREST(config *restconfig.Config) *REST {
	return &REST{
		stashClient: versioned.NewForConfigOrDie(config),
		kubeClient:  kubernetes.NewForConfigOrDie(config),
		config:      config,
		convertor: NewCustomTableConvertor(schema.GroupResource{
			Group:    repov1alpha1.SchemeGroupVersion.Group,
			Resource: repov1alpha1.ResourcePluralSnapshot,
		}),
	}
}

func (r *REST) NamespaceScoped() bool {
	return true
}

func (r *REST) New() runtime.Object {
	return &repositories.Snapshot{}
}

func (r *REST) Destroy() {}

func (r *REST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return repov1alpha1.SchemeGroupVersion.WithKind(repov1alpha1.ResourceKindSnapshot)
}

func (r *REST) Categories() []string {
	return []string{"storage", "appscode", "all"}
}

func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("missing namespace")
	}

	repoName, snapshotId, err := util.GetRepoNameAndSnapshotID(name)
	if err != nil {
		return nil, apierrors.NewBadRequest(err.Error())
	}

	repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(ctx, repoName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, apierrors.NewNotFound(stash.Resource(stash.ResourceSingularRepository), repoName)
		}
		return nil, apierrors.NewInternalError(err)
	}

	secret, err := r.kubeClient.CoreV1().Secrets(repo.Namespace).Get(context.TODO(), repo.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, apierrors.NewNotFound(core.Resource("secret"), repo.Spec.Backend.StorageSecretName)
		}
		return nil, apierrors.NewInternalError(err)
	}
	opt := Options{
		Repository:  repo,
		Secret:      secret,
		SnapshotIDs: []string{snapshotId},
		InCluster:   false,
	}

	snapshots, err := r.GetSnapshotsFromBackned(opt)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	if len(snapshots) == 0 {
		return nil, apierrors.NewNotFound(repositories.Resource(repov1alpha1.ResourceSingularSnapshot), name)
	}

	// TODO: return &snapshots[0], nil
	snapshot := &snapshots[0]
	return snapshot, nil
}

func (r *REST) NewList() runtime.Object {
	return &repositories.SnapshotList{}
}

func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("missing namespace")
	}

	repos, err := r.stashClient.StashV1alpha1().Repositories(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	// filter by repository label
	var selectedRepos []stash.Repository
	if options.LabelSelector != nil && hasSelector(options.LabelSelector, KeyRepository) {
		for _, r := range repos.Items {
			repoLabels := map[string]string{
				KeyRepository: r.Name,
			}
			if r.Labels != nil {
				repoLabels = meta_util.OverwriteKeys(repoLabels, r.Labels)
			}
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
		secret, err := r.kubeClient.CoreV1().Secrets(repo.Namespace).Get(context.TODO(), repo.Spec.Backend.StorageSecretName, metav1.GetOptions{})
		if err != nil {
			if kerr.IsNotFound(err) {
				return nil, apierrors.NewNotFound(core.Resource("secret"), repo.Spec.Backend.StorageSecretName)
			}
			return nil, apierrors.NewInternalError(err)
		}
		opt := Options{
			Repository:  &repo,
			Secret:      secret,
			SnapshotIDs: nil,
			InCluster:   false,
		}
		snapshots, err := r.GetSnapshotsFromBackned(opt)
		if err != nil {
			return nil, apierrors.NewInternalError(err)
		}
		// filter by hostname label
		if options.LabelSelector != nil && hasSelector(options.LabelSelector, KeyHostname) {
			for i := range snapshots {
				hostLabels := map[string]string{
					KeyHostname: snapshots[i].Status.Hostname,
				}
				if options.LabelSelector.Matches(labels.Set(hostLabels)) {
					snapshotList.Items = append(snapshotList.Items, snapshots[i])
				}
			}
		} else {
			snapshotList.Items = append(snapshotList.Items, snapshots...)
		}
	}

	// k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructured_list.go
	// unstructured.UnstructuredList{}
	return snapshotList, nil
}

func (r *REST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return r.convertor.ConvertToTable(ctx, object, tableOptions)
}

func (r *REST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	ns, ok := apirequest.NamespaceFrom(ctx)
	if !ok {
		return nil, false, apierrors.NewBadRequest("missing namespace")
	}
	repoName, snapshotId, err := util.GetRepoNameAndSnapshotID(name)
	if err != nil {
		return nil, false, apierrors.NewBadRequest(err.Error())
	}
	repo, err := r.stashClient.StashV1alpha1().Repositories(ns).Get(ctx, repoName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, false, apierrors.NewNotFound(stash.Resource(stash.ResourceSingularRepository), repoName)
		}
		return nil, false, apierrors.NewInternalError(err)
	}

	secret, err := r.kubeClient.CoreV1().Secrets(repo.Namespace).Get(context.TODO(), repo.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			return nil, false, apierrors.NewNotFound(core.Resource("secret"), repo.Spec.Backend.StorageSecretName)
		}
		return nil, false, apierrors.NewInternalError(err)
	}
	opt := Options{
		Repository:  repo,
		Secret:      secret,
		SnapshotIDs: []string{snapshotId},
		InCluster:   false,
	}

	// first, check if the snapshot exist
	snapshots, err := r.GetSnapshotsFromBackned(opt)
	if err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}
	if len(snapshots) == 0 {
		return nil, false, apierrors.NewNotFound(repositories.Resource(repov1alpha1.ResourceSingularSnapshot), name)
	}
	// delete snapshot
	if err = r.ForgetSnapshotsFromBackend(opt); err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}
	return nil, true, nil
}

func hasSelector(selector labels.Selector, key string) bool {
	if selector == nil {
		return false
	}
	requirements, _ := selector.Requirements()
	for i := range requirements {
		if requirements[i].Key() == key {
			return true
		}
	}
	return false
}
