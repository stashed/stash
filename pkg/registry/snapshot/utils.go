package snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/appscode/go/log"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	core_util "kmodules.xyz/client-go/core/v1"
	"stash.appscode.dev/stash/apis/repositories"
	stash "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/pkg/cli"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"
)

const (
	ExecStash      = "/bin/stash"
	scratchDirName = "scratch"
	secretDirName  = "secret"
)

func (r *REST) getSnapshots(repository *stash.Repository, snapshotIDs []string) ([]repositories.Snapshot, error) {
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

func (r *REST) forgetSnapshots(repository *stash.Repository, snapshotIDs []string) error {
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
	return err
}

func (r *REST) execOnSidecar(repository *stash.Repository, cmd string, snapshotIDs []string) ([]byte, error) {
	// get workload name from repository
	var workloadName string
	if isV1Alpha1Repository(*repository) {
		info, err := util.ExtractDataFromRepositoryLabel(repository.Labels)
		if err != nil {
			return nil, err
		}
		workloadName = info.WorkloadName
	} else {
		// get backup config for repository
		bc, err := util.FindBackupConfigForRepository(r.stashClient, *repository)
		if err != nil {
			return nil, err
		}
		// only allow sidecar model
		if bc.Spec.Target == nil || util.BackupModel(bc.Spec.Target.Ref.Kind) == util.ModelCronJob {
			return nil, fmt.Errorf("can't list snapshots for loacl backend with backup model 'cronjob'")
		}
		workloadName = bc.Spec.Target.Ref.Name
	}

	// get pod for workload
	pod, err := r.getPodWithStashSidecar(repository.Namespace, workloadName)
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

	log.Infof("Executing command %v on pod %v", command, pod.Name)

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
		return nil, fmt.Errorf("could not execute: %v, reason: %s", err, execErr.String())
	}

	return execOut.Bytes(), nil
}

func (r *REST) getV1Beta1Snapshots(repository *stash.Repository, snapshotIDs []string) ([]repositories.Snapshot, error) {
	tempDir, err := ioutil.TempDir("", "stash")
	if err != nil {
		return nil, err
	}
	// cleanup whole tempDir dir at the end
	defer os.RemoveAll(tempDir)

	scratchDir := filepath.Join(tempDir, scratchDirName)
	secretDir := filepath.Join(tempDir, secretDirName)

	// get source repository secret
	secret, err := r.kubeClient.CoreV1().Secrets(repository.Namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

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

func (r *REST) forgetV1Beta1Snapshots(repository *stash.Repository, snapshotIDs []string) error {
	tempDir, err := ioutil.TempDir("", "stash")
	if err != nil {
		return err
	}
	// cleanup whole tempDir dir at the end
	defer os.RemoveAll(tempDir)

	scratchDir := filepath.Join(tempDir, scratchDirName)
	secretDir := filepath.Join(tempDir, secretDirName)

	// get source repository secret
	secret, err := r.kubeClient.CoreV1().Secrets(repository.Namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

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

func (r *REST) GetVersionedSnapshots(repository *stash.Repository, snapshotIDs []string, inCluster bool) ([]repositories.Snapshot, error) {
	if repository.Spec.Backend.Local != nil && !inCluster {
		return r.getSnapshotsFromSidecar(repository, snapshotIDs)
	} else if isV1Alpha1Repository(*repository) {
		return r.getSnapshots(repository, snapshotIDs)
	}
	return r.getV1Beta1Snapshots(repository, snapshotIDs)
}

func (r *REST) ForgetVersionedSnapshots(repository *stash.Repository, snapshotIDs []string, inCluster bool) error {
	if repository.Spec.Backend.Local != nil && !inCluster {
		return r.forgetSnapshotsFromSidecar(repository, snapshotIDs)
	} else if isV1Alpha1Repository(*repository) {
		return r.forgetSnapshots(repository, snapshotIDs)
	}
	return r.forgetV1Beta1Snapshots(repository, snapshotIDs)
}

// v1alpha1 repositories should have 'restic' label
func isV1Alpha1Repository(repository stash.Repository) bool {
	if repository.Labels == nil {
		return false
	}
	_, ok := repository.Labels["restic"]
	return ok
}
