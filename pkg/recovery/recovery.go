/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package recovery

import (
	"context"
	"fmt"
	"time"

	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/client/clientset/versioned/scheme"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1"
	stash_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1/util"
	"stash.appscode.dev/stash/pkg/cli"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	"github.com/cenkalti/backoff"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/reference"
)

type Controller struct {
	k8sClient      kubernetes.Interface
	stashClient    cs.StashV1alpha1Interface
	namespace      string
	recoveryName   string
	backoffMaxWait time.Duration
}

const (
	RecoveryEventComponent = "stash-recovery"
)

func New(k8sClient kubernetes.Interface, stashClient cs.StashV1alpha1Interface, namespace, name string, backoffMaxWait time.Duration) *Controller {
	return &Controller{
		k8sClient:      k8sClient,
		stashClient:    stashClient,
		namespace:      namespace,
		recoveryName:   name,
		backoffMaxWait: backoffMaxWait,
	}
}

func (c *Controller) Run() {
	var recovery *api.Recovery
	var err error

	operation := func() error {
		recovery, err = c.stashClient.Recoveries(c.namespace).Get(context.TODO(), c.recoveryName, metav1.GetOptions{})
		return err
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = c.backoffMaxWait
	err = backoff.Retry(operation, b)

	if err != nil {
		log.Errorln(err)
		return
	}

	if err = recovery.IsValid(); err != nil {
		log.Errorf("Failed to validate recovery %s, reason: %s", recovery.Name, err)
		_, err = stash_util.UpdateRecoveryStatus(context.TODO(), c.stashClient, recovery.ObjectMeta, func(in *api.RecoveryStatus) *api.RecoveryStatus {
			in.Phase = api.RecoveryFailed
			return in
		}, metav1.UpdateOptions{})
		if err != nil {
			log.Errorln(err)
		}
		ref, rerr := reference.GetReference(scheme.Scheme, recovery)
		if rerr == nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				RecoveryEventComponent,
				ref,
				core.EventTypeWarning,
				eventer.EventReasonFailedToRecover,
				fmt.Sprintf("Failed to validate recovery %s, reason: %s", recovery.Name, err),
			)
		} else {
			log.Errorf("Failed to write event on %s %s. Reason: %s", recovery.Kind, recovery.Name, rerr)
		}
		return
	}

	if err = c.RecoverOrErr(recovery); err != nil {
		log.Errorf("Failed to complete recovery %s, reason: %s", recovery.Name, err)
		_, err = stash_util.UpdateRecoveryStatus(context.TODO(), c.stashClient, recovery.ObjectMeta, func(in *api.RecoveryStatus) *api.RecoveryStatus {
			in.Phase = api.RecoveryFailed
			return in
		}, metav1.UpdateOptions{})
		if err != nil {
			log.Errorln(err)
		}
		ref, rerr := reference.GetReference(scheme.Scheme, recovery)
		if rerr == nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				RecoveryEventComponent,
				ref,
				core.EventTypeWarning,
				eventer.EventReasonFailedToRecover,
				fmt.Sprintf("Failed to complete recovery %s, reason: %s", recovery.Name, err),
			)
		} else {
			log.Errorf("Failed to write event on %s %s. Reason: %s", recovery.Kind, recovery.Name, rerr)
		}
		return
	}

	log.Infof("Recovery %s succeeded\n", recovery.Name)
	_, err = stash_util.UpdateRecoveryStatus(context.TODO(), c.stashClient, recovery.ObjectMeta, func(in *api.RecoveryStatus) *api.RecoveryStatus {
		in.Phase = api.RecoverySucceeded
		// TODO: status.Stats
		return in
	}, metav1.UpdateOptions{})
	if err != nil {
		log.Errorln(err)
	}
	ref, rerr := reference.GetReference(scheme.Scheme, recovery)
	if rerr == nil {
		eventer.CreateEventWithLog(
			c.k8sClient,
			RecoveryEventComponent,
			ref,
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulRecovery,
			fmt.Sprintf("Recovery %s succeeded", recovery.Name),
		)
	} else {
		log.Errorf("Failed to write event on %s %s. Reason: %s", recovery.Kind, recovery.Name, rerr)
	}
}

func (c *Controller) RecoverOrErr(recovery *api.Recovery) error {
	repository, err := c.stashClient.Repositories(recovery.Spec.Repository.Namespace).Get(context.TODO(), recovery.Spec.Repository.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	repoLabelData, err := util.ExtractDataFromRepositoryLabel(repository.Labels)
	if err != nil {
		return err
	}
	secret, err := c.k8sClient.CoreV1().Secrets(repository.Namespace).Get(context.TODO(), repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	workload := &api.LocalTypedReference{
		Kind: repoLabelData.WorkloadKind,
		Name: repoLabelData.WorkloadName,
	}
	hostname, smartPrefix, err := workload.HostnamePrefix(repoLabelData.PodName, repoLabelData.NodeName)
	if err != nil {
		return err
	}

	backend := util.FixBackendPrefix(repository.Spec.Backend.DeepCopy(), smartPrefix)

	wrapper := cli.New("/tmp", false, hostname)
	if _, err = wrapper.SetupEnv(*backend, secret, smartPrefix); err != nil {
		return err
	}

	snapshotID := ""
	if recovery.Spec.Snapshot != "" {
		_, snapshotID, err = util.GetRepoNameAndSnapshotID(recovery.Spec.Snapshot)
		if err != nil {
			return err
		}
	}

	var errRec error
	for _, path := range recovery.Spec.Paths {
		d, err := c.measure(wrapper.Restore, path, hostname, snapshotID)
		if err != nil {
			errRec = err
			ref, rerr := reference.GetReference(scheme.Scheme, recovery)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					RecoveryEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedToRecover,
					fmt.Sprintf("failed to recover FileGroup %s, reason: %v", path, err),
				)
			} else {
				log.Errorf("Failed to write event on %s %s. Reason: %s", recovery.Kind, recovery.Name, rerr)
			}
			_, err = stash_util.SetRecoveryStats(context.TODO(), c.stashClient, recovery.ObjectMeta, path, d, api.RecoveryFailed, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		} else {
			_, err = stash_util.SetRecoveryStats(context.TODO(), c.stashClient, recovery.ObjectMeta, path, d, api.RecoverySucceeded, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	return errRec
}

func (c *Controller) measure(f func(string, string, string) error, path, host, snapshotID string) (time.Duration, error) {
	startTime := time.Now()
	err := f(path, host, snapshotID)
	return time.Since(startTime), err
}
