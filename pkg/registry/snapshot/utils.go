package snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	api "github.com/appscode/stash/apis/repositories/v1alpha1"
	"github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/cli"
	"github.com/appscode/stash/pkg/util"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	ExecStash                      = "/bin/stash"
	SnapshotIDLength               = 8
	SnapshotIDLengthWithDashPrefix = 9
)

type labelsInfo struct {
	workloadKind string
	workloadName string
	podName      string
	nodeName     string
}

func (r *REST) GetSnapshots(repository *v1alpha1.Repository, snapshotIDs []string) ([]api.Snapshot, error) {
	backend := repository.Spec.Backend.DeepCopy()

	info, err := extractDataFromRepositoryLabel(repository.Labels)
	if err != nil {
		return nil, err
	}

	workload := &v1alpha1.LocalTypedReference{
		Kind: info.workloadKind,
		Name: info.workloadName,
	}
	hostName, smartPrefix, err := workload.HostnamePrefix(info.podName, info.nodeName)

	secret, err := r.kubeClient.CoreV1().Secrets(repository.Namespace).Get(backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	backend = fixBackendPrefix(backend, smartPrefix)

	cli := cli.New("/tmp", false, hostName)
	if _, err = cli.SetupEnv(*backend, secret, smartPrefix); err != nil {
		return nil, err
	}

	results, err := cli.ListSnapshots(snapshotIDs)
	if err != nil {
		return nil, err
	}

	snapshots := make([]api.Snapshot, 0)
	snapshot := &api.Snapshot{}
	for _, result := range results {
		snapshot.Namespace = repository.Namespace
		snapshot.Name = repository.Name + "-" + result.ID[0:8] // snapshotName = repositoryName-first8CharacterOfSnapshotId
		snapshot.UID = types.UID(result.ID)

		snapshot.Labels = repository.Labels
		snapshot.Labels["repository"] = repository.Name

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

func (r *REST) ForgetSnapshots(repository *v1alpha1.Repository, snapshotIDs []string) error {
	backend := repository.Spec.Backend.DeepCopy()

	info, err := extractDataFromRepositoryLabel(repository.Labels)
	if err != nil {
		return err
	}

	workload := &v1alpha1.LocalTypedReference{
		Kind: info.workloadKind,
		Name: info.workloadName,
	}
	hostName, smartPrefix, err := workload.HostnamePrefix(info.podName, info.nodeName)

	secret, err := r.kubeClient.CoreV1().Secrets(repository.Namespace).Get(backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	backend = fixBackendPrefix(backend, smartPrefix)

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

func (r *REST) getSnapshotsFromSidecar(repository *v1alpha1.Repository, snapshotIDs []string) ([]api.Snapshot, error) {
	response, err := r.execOnSidecar(repository, "snapshots", snapshotIDs)
	if err != nil {
		return nil, err
	}

	snapshots := make([]api.Snapshot, 0)
	err = json.Unmarshal(response, &snapshots)
	if err != nil {
		return nil, err
	}

	return snapshots, nil
}

func (r *REST) forgetSnapshotsFromSidecar(repository *v1alpha1.Repository, snapshotIDs []string) error {
	_, err := r.execOnSidecar(repository, "forget", snapshotIDs)
	if err != nil {
		return err
	}

	return nil
}
func (r *REST) execOnSidecar(repository *v1alpha1.Repository, cmd string, snapshotIDs []string) ([]byte, error) {
	info, err := extractDataFromRepositoryLabel(repository.Labels)
	if err != nil {
		return nil, err
	}

	pod, err := r.getPodWithStashSidecar(repository.Namespace, info.workloadName)
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
func extractDataFromRepositoryLabel(labels map[string]string) (info labelsInfo, err error) {
	var ok bool
	info.workloadKind, ok = labels["workload-kind"]
	if !ok {
		return info, errors.New("workload-kind not found in repository labels")
	}

	info.workloadName, ok = labels["workload-name"]
	if !ok {
		return info, errors.New("workload-name not found in repository labels")
	}

	info.podName, ok = labels["pod-name"]
	if !ok {
		info.podName = ""
	}

	info.nodeName, ok = labels["node-name"]
	if !ok {
		info.nodeName = ""
	}
	return info, nil
}

func fixBackendPrefix(backend *v1alpha1.Backend, autoPrefix string) *v1alpha1.Backend {
	if backend.Local != nil {
		backend.Local.SubPath = strings.TrimSuffix(backend.Local.SubPath, autoPrefix)
		backend.Local.SubPath = strings.TrimSuffix(backend.Local.SubPath, "/")
	} else if backend.S3 != nil {
		backend.S3.Prefix = strings.TrimSuffix(backend.S3.Prefix, autoPrefix)
		backend.S3.Prefix = strings.TrimSuffix(backend.S3.Prefix, "/")
		backend.S3.Prefix = strings.TrimPrefix(backend.S3.Prefix, backend.S3.Bucket)
		backend.S3.Prefix = strings.TrimPrefix(backend.S3.Prefix, "/")
	} else if backend.GCS != nil {
		backend.GCS.Prefix = strings.TrimSuffix(backend.GCS.Prefix, autoPrefix)
		backend.GCS.Prefix = strings.TrimSuffix(backend.GCS.Prefix, "/")
	} else if backend.Azure != nil {
		backend.Azure.Prefix = strings.TrimSuffix(backend.Azure.Prefix, autoPrefix)
		backend.Azure.Prefix = strings.TrimSuffix(backend.Azure.Prefix, "/")
	} else if backend.Swift != nil {
		backend.Swift.Prefix = strings.TrimSuffix(backend.Swift.Prefix, autoPrefix)
		backend.Swift.Prefix = strings.TrimSuffix(backend.Swift.Prefix, "/")
	} else if backend.B2 != nil {
		backend.B2.Prefix = strings.TrimSuffix(backend.B2.Prefix, autoPrefix)
		backend.B2.Prefix = strings.TrimSuffix(backend.B2.Prefix, "/")
	}

	return backend
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

func GetRepoNameAndSnapshotID(snapshotName string) (repoName, snapshotId string, err error) {
	if len(snapshotName) < 9 {
		err = errors.New("invalid snapshot name")
		return
	}
	snapshotId = snapshotName[len(snapshotName)-SnapshotIDLength:]

	repoName = strings.TrimSuffix(snapshotName, snapshotName[len(snapshotName)-SnapshotIDLengthWithDashPrefix:])
	return
}
