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

package framework

import (
	"context"
	"fmt"
	"strconv"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	v1alpha1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"
	"stash.appscode.dev/stash/pkg/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gomodules.xyz/stow"
	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kmodules.xyz/client-go/tools/portforward"
	store "kmodules.xyz/objectstore-api/api/v1"
	"kmodules.xyz/objectstore-api/osm"
)

type KindMetaReplicas struct {
	Kind     string
	Meta     metav1.ObjectMeta
	Replicas int
}

func (f *Framework) EventuallyRepositoryCreated(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() bool {
		_, err := f.StashClient.StashV1alpha1().Repositories(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
		if err == nil && !kerr.IsNotFound(err) {
			return true
		}
		return false
	},
		WaitTimeOut,
		PullInterval,
	)
}

func (f *Framework) GetRepositories(kmr KindMetaReplicas) []*v1alpha1.Repository {
	repoNames := make([]string, 0)
	nodeName := f.GetNodeName(kmr.Meta)
	workload := v1alpha1.LocalTypedReference{Name: kmr.Meta.Name, Kind: kmr.Kind}
	switch kmr.Kind {
	case apis.KindDeployment, apis.KindReplicationController, apis.KindReplicaSet, apis.KindDaemonSet:
		repoNames = append(repoNames, workload.GetRepositoryCRDName("", nodeName))
	case apis.KindStatefulSet:
		for i := 0; i < kmr.Replicas; i++ {
			repoNames = append(repoNames, workload.GetRepositoryCRDName(kmr.Meta.Name+"-"+strconv.Itoa(i), nodeName))
		}
	}
	repositories := make([]*v1alpha1.Repository, 0)
	for _, repoName := range repoNames {
		obj, err := f.StashClient.StashV1alpha1().Repositories(kmr.Meta.Namespace).Get(context.TODO(), repoName, metav1.GetOptions{})
		if err == nil {
			repositories = append(repositories, obj)
		}
	}
	return repositories
}

func (f *Framework) DeleteRepository(repository *v1alpha1.Repository) error {
	err := f.StashClient.StashV1alpha1().Repositories(repository.Namespace).Delete(context.TODO(), repository.Name, metav1.DeleteOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}
func (f *Framework) BrowseBackendRepository(repository *v1alpha1.Repository) ([]stow.Item, error) {
	cfg, err := osm.NewOSMContext(f.KubeClient, repository.Spec.Backend, repository.Namespace)
	if err != nil {
		return nil, err
	}

	loc, err := stow.Dial(cfg.Provider, cfg.Config)
	if err != nil {
		return nil, err
	}

	bucket, prefix, err := util.GetBucketAndPrefix(&repository.Spec.Backend)
	if err != nil {
		return nil, err
	}
	prefix = prefix + "/"

	container, err := loc.Container(bucket)
	if err != nil {
		return nil, err
	}

	cursor := stow.CursorStart
	items, _, err := container.Items(prefix, cursor, 50)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// BrowseMinioRepository browse backend minio repository to check if there is any data in the backend.
// Minio server is running inside the cluster but the test is running outside of the cluster.
// So, we can't access the Minio server using the service created for it.
// Here, we are going to port-forward the Minio pod and and use the port-forwarded address to access the backend.
func (f *Framework) BrowseMinioRepository(repo *v1alpha1.Repository) ([]stow.Item, error) {
	if repo.Spec.Backend.S3 == nil {
		return nil, fmt.Errorf("failed to browse desired backend repository. Reason: Provided Repository does not use S3 or S3 compatible backend")
	}
	pod, err := f.GetMinioPod()
	if err != nil {
		return nil, err
	}

	tunnel := portforward.NewTunnel(portforward.TunnelOptions{
		Client:    f.KubeClient.CoreV1().RESTClient(),
		Config:    f.ClientConfig,
		Resource:  "pods",
		Name:      pod.Name,
		Namespace: f.namespace,
		Remote:    443,
	})
	defer tunnel.Close()

	err = tunnel.ForwardPort()
	if err != nil {
		return nil, err
	}

	// update endpoint so that BrowseBackendRepository() function uses the port-forwarded address
	repo.Spec.Backend.S3.Endpoint = fmt.Sprintf("https://%s:%d", LocalHostIP, tunnel.Local)
	return f.BrowseBackendRepository(repo)
}

func (fi *Invocation) NewGCSRepository(secretName string, maxConnection int64) *v1alpha1.Repository {
	return &v1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("gcs-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: v1alpha1.RepositorySpec{
			Backend: store.Backend{
				GCS: &store.GCSSpec{
					Bucket:         "stash-ci",
					Prefix:         fmt.Sprintf("stash-e2e/%s/%s", fi.namespace, fi.app),
					MaxConnections: maxConnection,
				},
				StorageSecretName: secretName,
			},
			WipeOut: true,
		},
	}
}

func (fi *Invocation) NewMinioRepository(secretName string) *v1alpha1.Repository {
	return &v1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("minio-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: v1alpha1.RepositorySpec{
			Backend: store.Backend{
				S3: &store.S3Spec{
					Endpoint: fi.MinioServiceAddres(),
					Bucket:   fi.app,
					Prefix:   fmt.Sprintf("stash-e2e/%s/%s", fi.namespace, fi.app),
				},
				StorageSecretName: secretName,
			},
			WipeOut: false,
		},
	}
}

func (fi *Invocation) NewS3Repository(secretName string) *v1alpha1.Repository {
	return &v1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("s3-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: v1alpha1.RepositorySpec{
			Backend: store.Backend{
				S3: &store.S3Spec{
					Endpoint: "s3.amazonaws.com",
					Bucket:   "appscode-qa",
					Prefix:   fmt.Sprintf("stash-e2e/%s/%s", fi.namespace, fi.app),
				},
				StorageSecretName: secretName,
			},
			WipeOut: true,
		},
	}
}

func (fi *Invocation) NewAzureRepository(secretName string, maxConnection int64) *v1alpha1.Repository {
	return &v1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("azure-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: v1alpha1.RepositorySpec{
			Backend: store.Backend{
				Azure: &store.AzureSpec{
					Container:      "appscode-qa",
					Prefix:         fmt.Sprintf("stash-e2e/%s/%s", fi.namespace, fi.app),
					MaxConnections: maxConnection,
				},
				StorageSecretName: secretName,
			},
			WipeOut: true,
		},
	}
}

func (fi *Invocation) NewRestRepository(tls bool, secretName string) *v1alpha1.Repository {
	scheme := "http"
	if tls {
		scheme = "https"
	}
	return &v1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("rest-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: v1alpha1.RepositorySpec{
			Backend: store.Backend{
				Rest: &store.RestServerSpec{
					URL: fmt.Sprintf("%s://%s:8000/myuser", scheme, fi.RestServiceAddres()),
				},
				StorageSecretName: secretName,
			},
		},
	}
}

func (fi *Invocation) NewSwiftRepository(secretName string) *v1alpha1.Repository {
	return &v1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("swift-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: v1alpha1.RepositorySpec{
			Backend: store.Backend{
				Swift: &store.SwiftSpec{
					Container: "stash-backup",
					Prefix:    fmt.Sprintf("stash-e2e/%s/%s", fi.namespace, fi.app),
				},
				StorageSecretName: secretName,
			},
			WipeOut: true,
		},
	}
}

func (fi *Invocation) NewB2Repository(secretName string, maxConnection int64) *v1alpha1.Repository {
	return &v1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("b2-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: v1alpha1.RepositorySpec{
			Backend: store.Backend{
				B2: &store.B2Spec{
					Bucket:         "stash-qa",
					Prefix:         fmt.Sprintf("stash-e2e/%s/%s", fi.namespace, fi.app),
					MaxConnections: maxConnection,
				},
				StorageSecretName: secretName,
			},
		},
	}
}

func (fi *Invocation) SetupGCSRepository(maxConnection int64, appendRepoToCleanupList bool, transformFuncs ...func(repo *v1alpha1.Repository)) (*v1alpha1.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForGCSBackend()

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing GCS credential")
	}
	_, err := fi.CreateSecret(cred)
	if err != nil {
		return nil, err
	}

	// Generate Repository Definition
	repo := fi.NewGCSRepository(cred.Name, maxConnection)

	// transformFuncs provides a array of functions that made test specific change on the Repository
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(repo)
	}
	// Create Repository
	By("Creating Repository")
	repo, err = fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return repo, err
	}

	// If `.spec.wipeOut` is set to "true", then the corresponding Secret is required to delete the remote repository.
	// Hence we need to delete the Repository object first.
	if appendRepoToCleanupList {
		fi.AppendToCleanupList(repo)
	}
	fi.AppendToCleanupList(&cred)

	return repo, nil
}

func (fi *Invocation) SetupMinioRepository(transformFuncs ...func(repo *v1alpha1.Repository)) (*v1alpha1.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForMinioBackend(true)

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing Minio credential")
	}
	_, err := fi.CreateSecret(cred)
	if err != nil {
		return nil, err
	}

	// Generate Repository Definition
	repo := fi.NewMinioRepository(cred.Name)

	// transformFuncs provides a array of functions that made test specific change on the Repository
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(repo)
	}
	return fi.CreateRepository(repo, &cred)
}

func (fi *Invocation) CreateRepository(repo *v1alpha1.Repository, cred *core.Secret) (*v1alpha1.Repository, error) {
	By("Creating Repository")
	repo, err := fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return repo, err
	}

	// If `.spec.wipeOut` is set to "true", then the corresponding Secret is required to delete the remote repository.
	// Hence we need to delete the Repository object first.
	fi.AppendToCleanupList(repo, cred)
	return repo, nil
}

func (fi *Invocation) SetupS3Repository(appendToCleanupList bool, transformFuncs ...func(repo *v1alpha1.Repository)) (*v1alpha1.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForS3Backend()

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing S3 credential")
	}
	_, err := fi.CreateSecret(cred)
	if err != nil {
		return nil, err
	}

	// Generate Repository Definition
	repo := fi.NewS3Repository(cred.Name)

	// transformFuncs provides a array of functions that made test specific change on the Repository
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(repo)
	}
	// Create Repository
	By("Creating Repository")
	repo, err = fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return repo, err
	}

	// If `.spec.wipeOut` is set to "true", then the corresponding Secret is required to delete the remote repository.
	// Hence we need to delete the Repository object first.
	if appendToCleanupList {
		fi.AppendToCleanupList(repo)
	}
	fi.AppendToCleanupList(&cred)

	return repo, nil
}

func (fi *Invocation) SetupAzureRepository(maxConnection int64, addRepoToCleanupList bool, transformFuncs ...func(repo *v1alpha1.Repository)) (*v1alpha1.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForAzureBackend()

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing Azure credential")
	}
	_, err := fi.CreateSecret(cred)
	if err != nil {
		return nil, err
	}

	// Generate Repository Definition
	repo := fi.NewAzureRepository(cred.Name, maxConnection)

	// transformFuncs provides a array of functions that made test specific change on the Repository
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(repo)
	}
	// Create Repository
	By("Creating Repository")
	repo, err = fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return repo, err
	}

	// If `.spec.wipeOut` is set to "true", then the corresponding Secret is required to delete the remote repository.
	// Hence we need to delete the Repository object first.
	if addRepoToCleanupList {
		fi.AppendToCleanupList(repo)
	}
	fi.AppendToCleanupList(&cred)

	return repo, nil
}

func (fi *Invocation) SetupRestRepository(tls bool, username, password string, transformFuncs ...func(repo *v1alpha1.Repository)) (*v1alpha1.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForRestBackend(tls, username, password)

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing Rest credential")
	}
	_, err := fi.CreateSecret(cred)
	if err != nil {
		return nil, err
	}

	// Generate Repository Definition
	repo := fi.NewRestRepository(tls, cred.Name)

	// transformFuncs provides a array of functions that made test specific change on the Repository
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(repo)
	}
	// Create Repository
	By("Creating Repository")
	repo, err = fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return repo, err
	}

	// If `.spec.wipeOut` is set to "true", then the corresponding Secret is required to delete the remote repository.
	// Hence we need to delete the Repository object first.
	fi.AppendToCleanupList(repo)
	fi.AppendToCleanupList(&cred)

	return repo, nil
}

func (fi *Invocation) SetupSwiftRepository(appendRepoToCleanupList bool, transformFuncs ...func(repo *v1alpha1.Repository)) (*v1alpha1.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForSwiftBackend()

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing Rest credential")
	}
	_, err := fi.CreateSecret(cred)
	if err != nil {
		return nil, err
	}

	// Generate Repository Definition
	repo := fi.NewSwiftRepository(cred.Name)

	// transformFuncs provides a array of functions that made test specific change on the Repository
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(repo)
	}
	// Create Repository
	By("Creating Repository")
	repo, err = fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return repo, err
	}

	// If `.spec.wipeOut` is set to "true", then the corresponding Secret is required to delete the remote repository.
	// Hence we need to delete the Repository object first.
	if appendRepoToCleanupList {
		fi.AppendToCleanupList(repo)
	}
	fi.AppendToCleanupList(&cred)

	return repo, nil
}

func (fi *Invocation) SetupB2Repository(maxConnection int64, transformFuncs ...func(repo *v1alpha1.Repository)) (*v1alpha1.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForB2Backend()

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing Rest credential")
	}
	_, err := fi.CreateSecret(cred)
	if err != nil {
		return nil, err
	}

	// Generate Repository Definition
	repo := fi.NewB2Repository(cred.Name, maxConnection)

	// transformFuncs provides a array of functions that made test specific change on the Repository
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(repo)
	}
	// Create Repository
	By("Creating Repository")
	repo, err = fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return repo, err
	}

	// If `.spec.wipeOut` is set to "true", then the corresponding Secret is required to delete the remote repository.
	// Hence we need to delete the Repository object first.
	fi.AppendToCleanupList(repo)
	fi.AppendToCleanupList(&cred)

	return repo, nil
}

func (fi *Invocation) AllowNamespace(repo *v1alpha1.Repository, allowedNamespace v1alpha1.FromNamespaces) (*v1alpha1.Repository, error) {
	repo, _, err := v1alpha1_util.PatchRepository(context.TODO(), fi.StashClient.StashV1alpha1(), repo, func(repository *v1alpha1.Repository) *v1alpha1.Repository {
		repository.Spec.UsagePolicy = &v1alpha1.UsagePolicy{
			AllowedNamespaces: v1alpha1.AllowedNamespaces{
				From: &allowedNamespace,
			},
		}
		return repository
	}, metav1.PatchOptions{})
	return repo, err
}
