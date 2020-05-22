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

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
	meta_util "kmodules.xyz/client-go/meta"
)

func (fi *Invocation) ReplicationController(name, pvcName, volName string) core.ReplicationController {
	name = rand.WithUniqSuffix(fmt.Sprintf("%s-%s", name, fi.app))
	labels := map[string]string{
		"app":  name,
		"kind": "replicationcontroller",
	}
	podTemplate := fi.PodTemplate(labels, pvcName, volName)
	return core.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: core.ReplicationControllerSpec{
			Replicas: types.Int32P(1),
			Template: &podTemplate,
		},
	}
}

func (f *Framework) CreateReplicationController(obj core.ReplicationController) (*core.ReplicationController, error) {
	return f.KubeClient.CoreV1().ReplicationControllers(obj.Namespace).Create(context.TODO(), &obj, metav1.CreateOptions{})
}

func (f *Framework) DeleteReplicationController(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().ReplicationControllers(meta.Namespace).Delete(context.TODO(), meta.Name, meta_util.DeleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyReplicationController(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *core.ReplicationController {
		obj, err := f.KubeClient.CoreV1().ReplicationControllers(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

func (fi *Invocation) WaitUntilRCReadyWithSidecar(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := fi.KubeClient.CoreV1().ReplicationControllers(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.Replicas == obj.Status.ReadyReplicas {
				pods, err := fi.GetAllPods(obj.ObjectMeta)
				if err != nil {
					return false, err
				}

				for i := range pods {
					hasSidecar := false
					for _, c := range pods[i].Spec.Containers {
						if c.Name == apis.StashContainer {
							hasSidecar = true
						}
					}
					if !hasSidecar {
						return false, nil
					}
				}
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
}

func (fi *Invocation) WaitUntilRCReadyWithInitContainer(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := fi.KubeClient.CoreV1().ReplicationControllers(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.Replicas == obj.Status.ReadyReplicas {
				pods, err := fi.GetAllPods(obj.ObjectMeta)
				if err != nil {
					return false, err
				}

				for i := range pods {
					hasInitContainer := false
					for _, c := range pods[i].Spec.InitContainers {
						if c.Name == apis.StashInitContainer {
							hasInitContainer = true
						}
					}
					if !hasInitContainer {
						return false, nil
					}
				}
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
}

func (fi *Invocation) DeployReplicationController(name string, replica int32, volName string) (*core.ReplicationController, error) {
	// append test case specific suffix so that name does not conflict during parallel test
	pvcName := fmt.Sprintf("%s-%s", volName, fi.app)

	// If the PVC does not exist, create PVC for Deployment
	pvc, err := fi.KubeClient.CoreV1().PersistentVolumeClaims(fi.namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			pvc, err = fi.CreateNewPVC(pvcName)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Generate ReplicationController definition
	rc := fi.ReplicationController(name, pvc.Name, volName)
	rc.Spec.Replicas = &replica

	By("Deploying ReplicationController: " + rc.Name)
	createdRC, err := fi.CreateReplicationController(rc)
	if err != nil {
		return createdRC, err
	}
	fi.AppendToCleanupList(createdRC)

	By("Waiting for ReplicationController to be ready")
	err = util.WaitUntilRCReady(fi.KubeClient, createdRC.ObjectMeta)
	Expect(err).NotTo(HaveOccurred())
	// check that we can execute command to the pod.
	// this is necessary because we will exec into the pods and create sample data
	fi.EventuallyAllPodsAccessible(createdRC.ObjectMeta).Should(BeTrue())

	return createdRC, err
}
