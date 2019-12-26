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
	"fmt"

	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
	apps_util "kmodules.xyz/client-go/apps/v1"
)

func (fi *Invocation) ReplicaSet(pvcName, volName string) apps.ReplicaSet {
	labels := map[string]string{
		"app":  fi.app,
		"kind": "replicaset",
	}
	return apps.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: types.Int32P(1),
			Template: fi.PodTemplate(labels, pvcName, volName),
		},
	}
}

func (f *Framework) CreateReplicaSet(obj apps.ReplicaSet) (*apps.ReplicaSet, error) {
	return f.KubeClient.AppsV1().ReplicaSets(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteReplicaSet(meta metav1.ObjectMeta) error {
	err := f.KubeClient.AppsV1().ReplicaSets(meta.Namespace).Delete(meta.Name, deleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyReplicaSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.ReplicaSet {
		obj, err := f.KubeClient.AppsV1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

func (f *Invocation) WaitUntilRSReadyWithSidecar(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.AppsV1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.Replicas == obj.Status.ReadyReplicas {
				pods, err := f.GetAllPods(obj.ObjectMeta)
				if err != nil {
					return false, err
				}

				for i := range pods {
					hasSidecar := false
					for _, c := range pods[i].Spec.Containers {
						if c.Name == util.StashContainer {
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

func (f *Invocation) WaitUntilRSReadyWithInitContainer(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.AppsV1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.Replicas == obj.Status.ReadyReplicas {
				pods, err := f.GetAllPods(obj.ObjectMeta)
				if err != nil {
					return false, err
				}

				for i := range pods {
					hasInitContainer := false
					for _, c := range pods[i].Spec.InitContainers {
						if c.Name == util.StashInitContainer {
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

func (f *Invocation) DeployReplicaSet(name string, replica int32, volName string) (*apps.ReplicaSet, error) {
	// append test case specific suffix so that name does not conflict during parallel test
	name = fmt.Sprintf("%s-%s", name, f.app)
	pvcName := fmt.Sprintf("%s-%s", volName, f.app)

	// If the PVC does not exist, create PVC for ReplicaSet
	pvc, err := f.KubeClient.CoreV1().PersistentVolumeClaims(f.namespace).Get(pvcName, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) {
			pvc, err = f.CreateNewPVC(pvcName)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Generate ReplicaSet definition
	rs := f.ReplicaSet(pvc.Name, volName)
	rs.Spec.Replicas = &replica
	rs.Name = name

	By("Deploying ReplicaSet: " + rs.Name)
	createdRS, err := f.CreateReplicaSet(rs)
	if err != nil {
		return createdRS, err
	}
	f.AppendToCleanupList(createdRS)

	By("Waiting for ReplicaSet to be ready")
	err = apps_util.WaitUntilReplicaSetReady(f.KubeClient, createdRS.ObjectMeta)
	Expect(err).NotTo(HaveOccurred())
	// check that we can execute command to the pod.
	// this is necessary because we will exec into the pods and create sample data
	f.EventuallyPodAccessible(createdRS.ObjectMeta).Should(BeTrue())

	return createdRS, err
}
