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
	"bufio"
	"fmt"
	"os"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/repositories"
	stash "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	meta_util "kmodules.xyz/client-go/meta"
)

const ExecStash = "/stash-enterprise"

type Options struct {
	Repository  *stash.Repository
	Secret      *core.Secret
	SnapshotIDs []string
	InCluster   bool
}

func (r *REST) GetSnapshotsFromBackned(opt Options) ([]repositories.Snapshot, error) {
	if opt.Repository.Spec.Backend.Local != nil && !opt.InCluster {
		return nil, fmt.Errorf("local backend isn't supported in Stash community edition")
	}
	return r.getSnapshotsFromBackend(opt)
}

func (r *REST) getSnapshotsFromBackend(opt Options) ([]repositories.Snapshot, error) {
	tempDir, err := os.MkdirTemp("", "stash")
	if err != nil {
		return nil, err
	}
	// cleanup whole tempDir dir at the end
	defer os.RemoveAll(tempDir)

	// configure restic wrapper
	extraOpt := &util.ExtraOptions{
		StorageSecret: opt.Secret,
		EnableCache:   false,
		ScratchDir:    tempDir,
	}
	// configure setupOption
	setupOpt, err := util.SetupOptionsForRepository(*opt.Repository, *extraOpt)
	if err != nil {
		return nil, fmt.Errorf("setup option for repository failed, reason: %s", err)
	}

	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(setupOpt)
	if err != nil {
		return nil, err
	}
	// if repository does not exist in the backend, then nothing to list. Just return.
	if !resticWrapper.RepositoryAlreadyExist() {
		klog.Infof("unable to verify whether repository exist or not in the backend for Repository: %s/%s", opt.Repository.Namespace, opt.Repository.Name)
		return nil, nil
	}
	// list snapshots, returns all snapshots for empty snapshotIDs
	// if there is no restic repository in the backend, this will return error.
	// in this case, we have to return empty snapshot list.
	results, err := resticWrapper.ListSnapshots(opt.SnapshotIDs)
	if err != nil {
		// check if the error is happening because of not having restic repository in the backend.
		if repoNotFound(resticWrapper.GetRepo(), err) {
			return nil, nil
		}
		return nil, err
	}

	snapshots := make([]repositories.Snapshot, 0)
	snapshot := &repositories.Snapshot{}
	for _, result := range results {
		snapshot.Namespace = opt.Repository.Namespace
		snapshot.Name = meta_util.NameWithSuffix(opt.Repository.Name, result.ID[0:apis.SnapshotIDLength]) // snapshotName = repositoryName-first8CharacterOfSnapshotId
		snapshot.UID = types.UID(result.ID)

		snapshot.Labels = map[string]string{
			"repository": opt.Repository.Name,
			"hostname":   result.Hostname,
		}
		if opt.Repository.Labels != nil {
			snapshot.Labels = meta_util.OverwriteKeys(snapshot.Labels, opt.Repository.Labels)
		}

		snapshot.CreationTimestamp.Time = result.Time
		snapshot.Status.UID = result.UID
		snapshot.Status.Gid = result.Gid
		snapshot.Status.Hostname = result.Hostname
		snapshot.Status.Paths = result.Paths
		snapshot.Status.Tree = result.Tree
		snapshot.Status.Username = result.Username
		snapshot.Status.Tags = result.Tags
		snapshot.Status.Repository = opt.Repository.Name

		snapshots = append(snapshots, *snapshot)
	}
	return snapshots, nil
}

func (r *REST) ForgetSnapshotsFromBackend(opt Options) error {
	if opt.Repository.Spec.Backend.Local != nil {
		return fmt.Errorf("local backend is not supported by Stash community edition")
	}
	return r.forgetSnapshotsFromBackend(opt)
}

func (r *REST) forgetSnapshotsFromBackend(opt Options) error {
	tempDir, err := os.MkdirTemp("", "stash")
	if err != nil {
		return err
	}
	// cleanup whole tempDir dir at the end
	defer os.RemoveAll(tempDir)

	// configure restic wrapper
	extraOpt := util.ExtraOptions{
		StorageSecret: opt.Secret,
		EnableCache:   false,
		ScratchDir:    tempDir,
	}
	// configure setupOption
	setupOpt, err := util.SetupOptionsForRepository(*opt.Repository, extraOpt)
	if err != nil {
		return fmt.Errorf("setup option for repository failed, reason: %s", err)
	}

	// init restic wrapper
	resticWrapper, err := restic.NewResticWrapper(setupOpt)
	if err != nil {
		return err
	}
	// delete snapshots
	_, err = resticWrapper.DeleteSnapshots(opt.SnapshotIDs)
	return err
}

func repoNotFound(repo string, err error) bool {
	repoNotFoundMessage := fmt.Sprintf("exit status 1, reason: %s", repo)

	scanner := bufio.NewScanner(strings.NewReader(err.Error()))
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		if strings.TrimSpace(line) == repoNotFoundMessage {
			return true
		}
	}
	return false
}
