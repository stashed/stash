package snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/appscode/stash/apis/repositories"
	stash "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/cli"
	"github.com/appscode/stash/pkg/restic"
	"github.com/appscode/stash/pkg/util"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	core_util "kmodules.xyz/client-go/core/v1"
)

const (
	ExecStash = "/bin/stash"
)

func (r *REST) GetSnapshots(repository *stash.Repository, snapshotIDs []string) ([]repositories.Snapshot, error) {
	backend := repository.Spec.Backend.DeepCopy()

	info, err := util.ExtractDataFromRepositoryLabel(repository.Labels)
	if err != nil {
		return nil, err
	}

	workload := &stash.LocalTypedReference{
		Kind: info.WorkloadKind,
		Name: info.WorkloadName,
	}
	hostName, smartPrefix, err := workload.HostnamePrefix(info.PodName, info.NodeName)

	secret, err := r.kubeClient.CoreV1().Secrets(repository.Namespace).Get(backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	backend = util.FixBackendPrefix(backend, smartPrefix)

	cli := cli.New("/tmp", false, hostName)
	if _, err = cli.SetupEnv(*backend, secret, smartPrefix); err != nil {
		return nil, err
	}

	results, err := cli.ListSnapshots(snapshotIDs)
	if err != nil {
		return nil, err
	}

	snapshots := make([]repositories.Snapshot, 0)
	snapshot := &repositories.Snapshot{}
	for _, result := range results {
		snapshot.Namespace = repository.Namespace
		snapshot.Name = repository.Name + "-" + result.ID[0:util.SnapshotIDLength] // snapshotName = repositoryName-first8CharacterOfSnapshotId
		snapshot.UID = types.UID(result.ID)

		snapshot.Labels = map[string]string{
			"repository": repository.Name,
		}
		if repository.Labels != nil {
			snapshot.Labels = core_util.UpsertMap(snapshot.Labels, repository.Labels)
		}

		snapshot.CreationTimestamp.Time = result.Time
		snapshot.Status.UID = result.UID
		snapshot.Status.Gid = result.Gid
		snapshot.Status.Hostname = result.Hostname
		snapshot.Status.Paths = result.Paths
		snapshot.Status.Tree = result.Tree
		snapshot.Status.Username = result.Username
		snapshot.Status.Tags = result.Tags

		snapshots = append(snapshots, *snapshot)
	}
	return snapshots, nil
}

func (r *REST) ForgetSnapshots(repository *stash.Repository, snapshotIDs []string) error {
	backend := repository.Spec.Backend.DeepCopy()

	info, err := util.ExtractDataFromRepositoryLabel(repository.Labels)
	if err != nil {
		return err
	}

	workload := &stash.LocalTypedReference{
		Kind: info.WorkloadKind,
		Name: info.WorkloadName,
	}
	hostName, smartPrefix, err := workload.HostnamePrefix(info.PodName, info.NodeName)

	secret, err := r.kubeClient.CoreV1().Secrets(repository.Namespace).Get(backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	backend = util.FixBackendPrefix(backend, smartPrefix)

	cli := cli.New("/tmp", false, hostName)
	if _, err = cli.SetupEnv(*backend, secret, smartPrefix); err != nil {
		return err
	}

	err = cli.DeleteSnapshots(snapshotIDs)
	if err != nil {
		return err
	}

	return nil
}

func (r *REST) getSnapshotsFromSidecar(repository *stash.Repository, snapshotIDs []string) ([]repositories.Snapshot, error) {
	response, err := r.execOnSidecar(repository, "snapshots", snapshotIDs)
	if err != nil {
		return nil, err
	}

	snapshots := make([]repositories.Snapshot, 0)
	err = json.Unmarshal(response, &snapshots)
	if err != nil {
		return nil, err
	}

	return snapshots, nil
}

func (r *REST) forgetSnapshotsFromSidecar(repository *stash.Repository, snapshotIDs []string) error {
	_, err := r.execOnSidecar(repository, "forget", snapshotIDs)
	if err != nil {
		return err
	}

	return nil
}
func (r *REST) execOnSidecar(repository *stash.Repository, cmd string, snapshotIDs []string) ([]byte, error) {
	info, err := util.ExtractDataFromRepositoryLabel(repository.Labels)
	if err != nil {
		return nil, err
	}

	pod, err := r.getPodWithStashSidecar(repository.Namespace, info.WorkloadName)
	if err != nil {
		return nil, err
	}

	command := []string{ExecStash, cmd, "--repo-name=" + repository.Name}
	if snapshotIDs != nil {
		command = append(command, snapshotIDs...)
	}

	response, err := r.execCommandOnPod(pod, command)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *REST) getPodWithStashSidecar(namespace, workloadname string) (*core.Pod, error) {
	podList, err := r.kubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pod := range podList.Items {
		if bytes.HasPrefix([]byte(pod.Name), []byte(workloadname)) && r.hasStashSidecar(&pod) {
			return &pod, nil
		}
	}

	return nil, fmt.Errorf("no pod found for workload %v", workloadname)
}

func (r *REST) hasStashSidecar(pod *core.Pod) bool {
	for _, c := range pod.Spec.Containers {
		if c.Name == util.StashContainer {
			return true
		}
	}
	return false
}

func (r *REST) execCommandOnPod(pod *core.Pod, command []string) ([]byte, error) {
	var (
		execOut bytes.Buffer
		execErr bytes.Buffer
	)

	req := r.kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")
	req.VersionedParams(&core.PodExecOptions{
		Container: util.StashContainer,
		Command:   command,
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(r.config, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to init executor: %v", err)
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &execOut,
		Stderr: &execErr,
		Tty:    true,
	})

	if err != nil {
		return nil, fmt.Errorf("could not execute: %v", err)
	}

	if execErr.Len() > 0 {
		return nil, fmt.Errorf("stderr: %v", execErr.String())
	}

	return execOut.Bytes(), nil
}

func (r *REST) GetV1Beta1Snapshots(repository *stash.Repository, snapshotIDs []string) ([]repositories.Snapshot, error) {
	var (
		scratchDir = "/tmp/stash-snapshots/scratch"
		secretDir  = "/tmp/stash-snapshots/secret"
	)

	// unlock local backend
	if repository.Spec.Backend.Local != nil {
		return nil, fmt.Errorf("can't get snapshots from repository with local backend")
	}

	// get source repository secret
	secret, err := r.kubeClient.CoreV1().Secrets(repository.Namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// cleanup whole scratch/secret dir at the end
	defer os.RemoveAll(scratchDir)
	defer os.RemoveAll(secretDir)

	// write repository secrets in a temp dir
	if err := os.MkdirAll(secretDir, 0755); err != nil {
		return nil, err
	}
	for key, value := range secret.Data {
		if err := ioutil.WriteFile(filepath.Join(secretDir, key), value, 0755); err != nil {
			return nil, err
		}
	}

	// configure restic wrapper
	extraOpt := util.ExtraOptions{
		SecretDir:   secretDir,
		EnableCache: false,
		ScratchDir:  scratchDir,
	}
	// configure setupOption
	setupOpt, err := util.SetupOptionsForRepository(*repository, extraOpt)
	if err != nil {
		return nil, fmt.Errorf("setup option for repository failed, reason: %s", err)
	}
	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(setupOpt)
	if err != nil {
		return nil, err
	}
	// list snapshots, returns all snapshots for empty snapshotIDs
	results, err := resticWrapper.ListSnapshots(snapshotIDs)
	if err != nil {
		return nil, err
	}

	snapshots := make([]repositories.Snapshot, 0)
	snapshot := &repositories.Snapshot{}
	for _, result := range results {
		snapshot.Namespace = repository.Namespace
		snapshot.Name = repository.Name + "-" + result.ID[0:util.SnapshotIDLength] // snapshotName = repositoryName-first8CharacterOfSnapshotId
		snapshot.UID = types.UID(result.ID)

		snapshot.Labels = map[string]string{
			"repository": repository.Name,
		}
		if repository.Labels != nil {
			snapshot.Labels = core_util.UpsertMap(snapshot.Labels, repository.Labels)
		}

		snapshot.CreationTimestamp.Time = result.Time
		snapshot.Status.UID = result.UID
		snapshot.Status.Gid = result.Gid
		snapshot.Status.Hostname = result.Hostname
		snapshot.Status.Paths = result.Paths
		snapshot.Status.Tree = result.Tree
		snapshot.Status.Username = result.Username
		snapshot.Status.Tags = result.Tags

		snapshots = append(snapshots, *snapshot)
	}
	return snapshots, nil
}

func (r *REST) DeleteV1Beta1Snapshots(repository *stash.Repository, snapshotIDs []string) error {
	var (
		scratchDir = "/tmp/stash-snapshots/scratch"
		secretDir  = "/tmp/stash-snapshots/secret"
	)

	// unlock local backend
	if repository.Spec.Backend.Local != nil {
		return fmt.Errorf("can't get snapshots from repository with local backend")
	}

	// get source repository secret
	secret, err := r.kubeClient.CoreV1().Secrets(repository.Namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// cleanup whole scratch/secret dir at the end
	defer os.RemoveAll(scratchDir)
	defer os.RemoveAll(secretDir)

	// write repository secrets in a temp dir
	if err := os.MkdirAll(secretDir, 0755); err != nil {
		return err
	}
	for key, value := range secret.Data {
		if err := ioutil.WriteFile(filepath.Join(secretDir, key), value, 0755); err != nil {
			return err
		}
	}

	// configure restic wrapper
	extraOpt := util.ExtraOptions{
		SecretDir:   secretDir,
		EnableCache: false,
		ScratchDir:  scratchDir,
	}
	// configure setupOption
	setupOpt, err := util.SetupOptionsForRepository(*repository, extraOpt)
	if err != nil {
		return fmt.Errorf("setup option for repository failed, reason: %s", err)
	}
	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(setupOpt)
	if err != nil {
		return err
	}
	// delete snapshots
	_, err = resticWrapper.DeleteSnapshots(snapshotIDs)
	return err
}

func (r *REST) getVersionedSnapshots(repository *stash.Repository, snapshotIDs []string) ([]repositories.Snapshot, error) {
	if isV1Alpha1Repository(*repository) {
		if repository.Spec.Backend.Local != nil {
			return r.getSnapshotsFromSidecar(repository, snapshotIDs)
		} else {
			return r.GetSnapshots(repository, snapshotIDs)
		}
	} else {
		return r.GetV1Beta1Snapshots(repository, snapshotIDs)
	}
}

func (r *REST) deleteVersionedSnapshots(repository *stash.Repository, snapshotIDs []string) error {
	if isV1Alpha1Repository(*repository) {
		if repository.Spec.Backend.Local != nil {
			return r.forgetSnapshotsFromSidecar(repository, snapshotIDs)
		} else {
			return r.ForgetSnapshots(repository, snapshotIDs)
		}
	} else {
		return r.DeleteV1Beta1Snapshots(repository, snapshotIDs)
	}
}

// v1alpha1 repositories should have 'restic' label
func isV1Alpha1Repository(repository stash.Repository) bool {
	if repository.Labels == nil {
		return false
	}
	_, ok := repository.Labels["restic"]
	return ok
}
