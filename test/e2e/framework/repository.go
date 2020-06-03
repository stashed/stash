/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

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
	"math"
	"strconv"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gomodules.xyz/stow"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_util "kmodules.xyz/client-go/meta"
	pfutil "kmodules.xyz/client-go/tools/portforward"
	store "kmodules.xyz/objectstore-api/api/v1"
	"kmodules.xyz/objectstore-api/osm"
)

type KindMetaReplicas struct {
	Kind     string
	Meta     metav1.ObjectMeta
	Replicas int
}

func (f *Framework) EventuallyRepository(workload interface{}) GomegaAsyncAssertion {
	return Eventually(func() []*api.Repository {
		switch w := workload.(type) {
		case *apps.DaemonSet:
			return f.DaemonSetRepos(w)
		case *apps.Deployment:
			return f.DeploymentRepos(w)
		case *core.ReplicationController:
			return f.ReplicationControllerRepos(w)
		case *apps.ReplicaSet:
			return f.ReplicaSetRepos(w)
		case *apps.StatefulSet:
			return f.StatefulSetRepos(w)
		default:
			return nil
		}
	})
}

func (f *Framework) EventuallyRepositoryCreated(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() bool {
		_, err := f.StashClient.StashV1alpha1().Repositories(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
		if err == nil && !kerr.IsNotFound(err) {
			return true
		}
		return false
	},
		time.Minute*2,
		time.Second*5,
	)
}

func (f *Framework) GetRepositories(kmr KindMetaReplicas) []*api.Repository {
	repoNames := make([]string, 0)
	nodeName := f.GetNodeName(kmr.Meta)
	workload := api.LocalTypedReference{Name: kmr.Meta.Name, Kind: kmr.Kind}
	switch kmr.Kind {
	case apis.KindDeployment, apis.KindReplicationController, apis.KindReplicaSet, apis.KindDaemonSet:
		repoNames = append(repoNames, workload.GetRepositoryCRDName("", nodeName))
	case apis.KindStatefulSet:
		for i := 0; i < kmr.Replicas; i++ {
			repoNames = append(repoNames, workload.GetRepositoryCRDName(kmr.Meta.Name+"-"+strconv.Itoa(i), nodeName))
		}
	}
	repositories := make([]*api.Repository, 0)
	for _, repoName := range repoNames {
		obj, err := f.StashClient.StashV1alpha1().Repositories(kmr.Meta.Namespace).Get(context.TODO(), repoName, metav1.GetOptions{})
		if err == nil {
			repositories = append(repositories, obj)
		}
	}
	return repositories
}

func (f *Framework) DeleteRepositories(repositories []*api.Repository) {
	for _, repo := range repositories {
		err := f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Delete(context.TODO(), repo.Name, meta_util.DeleteInBackground())
		Expect(err).NotTo(HaveOccurred())
	}
}

func (f *Framework) DeleteRepository(repository *api.Repository) error {
	err := f.StashClient.StashV1alpha1().Repositories(repository.Namespace).Delete(context.TODO(), repository.Name, metav1.DeleteOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}
func (f *Framework) BrowseBackendRepository(repository *api.Repository) ([]stow.Item, error) {
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
func (f *Framework) BrowseMinioRepository(repo *api.Repository) ([]stow.Item, error) {
	if repo.Spec.Backend.S3 == nil {
		return nil, fmt.Errorf("failed to browse desired backend repository. Reason: Provided Repository does not use S3 or S3 compatible backend")
	}
	pod, err := f.GetMinioPod()
	if err != nil {
		return nil, err
	}

	tunnel := pfutil.NewTunnel(f.KubeClient.CoreV1().RESTClient(), f.ClientConfig, f.namespace, pod.Name, 443)
	defer tunnel.Close()

	err = tunnel.ForwardPort()
	if err != nil {
		return nil, err
	}

	// update endpoint so that BrowseBackendRepository() function uses the port-forwarded address
	repo.Spec.Backend.S3.Endpoint = fmt.Sprintf("https://%s:%d", LocalHostIP, tunnel.Local)
	return f.BrowseBackendRepository(repo)
}

func (f *Framework) BackupCountInRepositoriesStatus(repos []*api.Repository) int64 {
	var backupCount int64 = math.MaxInt64

	// use minimum backupCount among all repos
	for _, repo := range repos {
		if int64(repo.Status.BackupCount) < backupCount {
			backupCount = int64(repo.Status.BackupCount)
		}
	}
	return backupCount
}

func (f *Framework) CreateRepository(repo *api.Repository) error {
	_, err := f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})

	return err

}

func (fi *Invocation) NewLocalRepositoryWithPVC(secretName string, pvcName string) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-local"),
			Namespace: fi.namespace,
		},
		Spec: api.RepositorySpec{
			Backend: store.Backend{
				Local: &store.LocalSpec{
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
					MountPath: TestSafeDataMountPath,
				},
				StorageSecretName: secretName,
			},
		},
	}
}

func (fi *Invocation) NewLocalRepositoryInHostPath(secretName string) *api.Repository {
	hostPathType := core.HostPathDirectoryOrCreate
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-local"),
			Namespace: fi.namespace,
		},
		Spec: api.RepositorySpec{
			Backend: store.Backend{
				Local: &store.LocalSpec{
					VolumeSource: core.VolumeSource{
						HostPath: &core.HostPathVolumeSource{
							Path: "/data/stash-test",
							Type: &hostPathType,
						},
					},
					MountPath: TestSafeDataMountPath,
				},
				StorageSecretName: secretName,
			},
		},
	}
}

func (fi *Invocation) NewLocalRepositoryInNFSServer(secretName string) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fi.app + "-local"),
			Namespace: fi.namespace,
		},
		Spec: api.RepositorySpec{
			Backend: store.Backend{
				Local: &store.LocalSpec{
					MountPath: TestSafeDataMountPath,
					VolumeSource: core.VolumeSource{
						NFS: &core.NFSVolumeSource{
							//Server: fmt.Sprintf("%s.%s.%s", nfsService, "cluster", "local"), // NFS server address
							Server: fi.GetNFSService(),
							Path:   "/", // this path is relative to "/exports" path of NFS server
						},
					},
				},
				StorageSecretName: secretName,
			},
		},
	}
}

func (fi *Invocation) NewGCSRepository(secretName string, maxConnection int64) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("gcs-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: api.RepositorySpec{
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

func (fi *Invocation) NewMinioRepository(secretName string) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("minio-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: api.RepositorySpec{
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

func (fi *Invocation) NewS3Repository(secretName string) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("s3-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: api.RepositorySpec{
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

func (fi *Invocation) NewAzureRepository(secretName string, maxConnection int64) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("azure-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: api.RepositorySpec{
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

func (fi *Invocation) NewRestRepository(secretName string) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("rest-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: api.RepositorySpec{
			Backend: store.Backend{
				Rest: &store.RestServerSpec{
					URL: fmt.Sprintf("http://%s:8000/myuser", fi.RestServiceAddres()),
				},
				StorageSecretName: secretName,
			},
		},
	}
}

func (fi *Invocation) NewSwiftRepository(secretName string) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("swift-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: api.RepositorySpec{
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

func (fi *Invocation) NewB2Repository(secretName string, maxConnection int64) *api.Repository {
	return &api.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(fmt.Sprintf("b2-%s", fi.app)),
			Namespace: fi.namespace,
		},
		Spec: api.RepositorySpec{
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

func (fi *Invocation) SetupLocalRepositoryWithPVC() (*api.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForLocalBackend()
	_, err := fi.CreateSecret(cred)
	Expect(err).NotTo(HaveOccurred())
	fi.AppendToCleanupList(&cred)

	// We are going to use a PVC as backend
	By("Creating Backend PVC")
	backendPVC := fi.PersistentVolumeClaim(rand.WithUniqSuffix("pvc"))
	backendPVC, err = fi.CreatePersistentVolumeClaim(backendPVC)
	Expect(err).NotTo(HaveOccurred())
	fi.AppendToCleanupList(backendPVC)

	// Generate Repository Definition
	repo := fi.NewLocalRepositoryWithPVC(cred.Name, backendPVC.Name)

	// Create Repository
	By("Creating Repository")
	repo, err = fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	fi.AppendToCleanupList(repo)

	return repo, nil
}

func (fi *Invocation) SetupLocalRepositoryWithHostPath() (*api.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForLocalBackend()
	_, err := fi.CreateSecret(cred)
	Expect(err).NotTo(HaveOccurred())
	fi.AppendToCleanupList(&cred)

	// We are going to use a hostPath as backend
	// Generate Repository Definition
	repo := fi.NewLocalRepositoryInHostPath(cred.Name)

	// Create Repository
	By("Creating Repository")
	repo, err = fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	fi.AppendToCleanupList(repo)

	return repo, nil
}

func (fi *Invocation) SetupLocalRepositoryWithNFSServer() (*api.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForLocalBackend()
	createdCred, err := fi.CreateSecret(cred)
	Expect(err).NotTo(HaveOccurred())
	fi.AppendToCleanupList(createdCred)

	// We are going to use a nfs server as backend
	// Generate Repository Definition
	repo := fi.NewLocalRepositoryInNFSServer(cred.Name)

	// Create Repository
	By("Creating Repository")
	repo, err = fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	fi.AppendToCleanupList(repo)

	return repo, nil
}

func (fi *Invocation) SetupGCSRepository(maxConnection int64, appendRepoToCleanupList bool) (*api.Repository, error) {
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

func (fi *Invocation) SetupMinioRepository() (*api.Repository, error) {
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

	// Create Repository
	By("Creating Repository")
	repo, err = fi.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(context.TODO(), repo, metav1.CreateOptions{})
	if err != nil {
		return repo, err
	}

	// If `.spec.wipeOut` is set to "true", then the corresponding Secret is required to delete the remote repository.
	// Hence we need to delete the Repository object first.
	fi.AppendToCleanupList(repo, &cred)

	return repo, nil
}

func (fi *Invocation) SetupS3Repository(appendToCleanupList bool) (*api.Repository, error) {
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

func (fi *Invocation) SetupAzureRepository(maxConnection int64, addRepoToCleanupList bool) (*api.Repository, error) {
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

func (fi *Invocation) SetupRestRepository(tls bool) (*api.Repository, error) {
	// Create Storage Secret
	By("Creating Storage Secret")
	cred := fi.SecretForRestBackend(tls)

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing Rest credential")
	}
	_, err := fi.CreateSecret(cred)
	if err != nil {
		return nil, err
	}

	// Generate Repository Definition
	repo := fi.NewRestRepository(cred.Name)

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

func (fi *Invocation) SetupSwiftRepository(appendRepoToCleanupList bool) (*api.Repository, error) {
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

func (fi *Invocation) SetupB2Repository(maxConnection int64) (*api.Repository, error) {
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
