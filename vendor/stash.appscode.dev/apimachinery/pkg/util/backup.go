/*
Copyright AppsCode Inc. and Contributors

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

package util

import (
	"context"
	"fmt"
	"time"

	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/conditions"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/restic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	cutil "kmodules.xyz/client-go/conditions"
)

type ActionOptions struct {
	StashClient       cs.Interface
	TargetRef         v1beta1.TargetRef
	SetupOptions      restic.SetupOptions
	BackupSessionName string
	Namespace         string
}

// ExecutePreBackupActions executes pre-backup actions such as InitializeBackendRepository etc.
func ExecutePreBackupActions(opt ActionOptions) error {
	backupSession, err := opt.StashClient.StashV1beta1().BackupSessions(opt.Namespace).Get(context.TODO(), opt.BackupSessionName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	session := invoker.NewBackupSessionHandler(opt.StashClient, backupSession)

	for _, targetStatus := range session.GetTargetStatus() {
		if invoker.TargetMatched(targetStatus.Ref, opt.TargetRef) && len(targetStatus.PreBackupActions) > 0 {
			return opt.executePreBackupActions(session, targetStatus)
		}
	}
	return nil
}

func (opt *ActionOptions) executePreBackupActions(session *invoker.BackupSessionHandler, targetStatus v1beta1.BackupTargetStatus) error {
	for _, action := range targetStatus.PreBackupActions {
		switch action {
		case v1beta1.InitializeBackendRepository:
			if !repoAlreadyInitialized(session) {
				err := opt.initializeBackendRepository(session)
				if err != nil {
					return conditions.SetBackendRepositoryInitializedConditionToFalse(session, err)
				}
			}
		default:
			return fmt.Errorf("unknown PreBackupAction: %s", action)
		}
	}
	return nil
}

func repoAlreadyInitialized(session *invoker.BackupSessionHandler) bool {
	return cutil.HasCondition(session.GetConditions(), v1beta1.BackendRepositoryInitialized)
}

// IsRepositoryInitialized check whether the backend restic repository has been initialized or not
func IsRepositoryInitialized(opt ActionOptions) (bool, error) {
	backupSession, err := opt.StashClient.StashV1beta1().BackupSessions(opt.Namespace).Get(context.Background(), opt.BackupSessionName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	// If the condition is not present, then the repository hasn't been initialized
	if !cutil.HasCondition(backupSession.Status.Conditions, v1beta1.BackendRepositoryInitialized) {
		return false, nil
	}
	// If the condition is present but it is set to "False", then the repository initialization has failed. Possibly due to invalid backend / storage secret.
	if !cutil.IsConditionTrue(backupSession.Status.Conditions, v1beta1.BackendRepositoryInitialized) {
		_, cnd := cutil.GetCondition(backupSession.Status.Conditions, v1beta1.BackendRepositoryInitialized)
		return false, fmt.Errorf(cnd.Reason)
	}
	return true, nil
}

func WaitForBackendRepository(opt ActionOptions) error {
	return wait.PollImmediate(5*time.Second, 30*time.Minute, func() (done bool, err error) {
		klog.Infof("Waiting for the backend repository.....")
		repoInitialized, err := IsRepositoryInitialized(opt)
		if err != nil {
			return false, err
		}
		// If the repository hasn't been initialized yet, it means some other process is responsible to initialize the repository.
		// So, retry after 5 seconds.
		if !repoInitialized {
			return false, nil
		}
		return true, nil
	})
}

func (opts *ActionOptions) initializeBackendRepository(session *invoker.BackupSessionHandler) error {
	w, err := restic.NewResticWrapper(opts.SetupOptions)
	if err != nil {
		return err
	}
	// initialize repository if it doesn't exist
	if !w.RepositoryAlreadyExist() {
		err = w.InitializeRepository()
		if err != nil {
			return err
		}
	}
	return conditions.SetBackendRepositoryInitializedConditionToTrue(session)
}
