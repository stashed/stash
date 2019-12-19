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
	"time"

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

const (
	ProberDemoPodPrefix = "prober-demo"
	ExitCodeSuccess     = "EXIT_CODE_SUCCESS"
	ExitCodeFail        = "EXIT_CODE_FAIL"
	HttpPortName        = "http-port"
	HttpPort            = 8080
	TcpPortName         = "tcp-port"
	TcpPort             = 9090
)

func (fi *Invocation) Deployment(pvcName string) apps.Deployment {
	labels := map[string]string{
		"app":  fi.app,
		"kind": "deployment",
	}
	return apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.DeploymentSpec{
			Replicas: types.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: fi.PodTemplate(labels, pvcName),
		},
	}
}

func (f *Framework) CreateDeployment(obj apps.Deployment) (*apps.Deployment, error) {
	return f.KubeClient.AppsV1().Deployments(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteDeployment(meta metav1.ObjectMeta) error {
	err := f.KubeClient.AppsV1().Deployments(meta.Namespace).Delete(meta.Name, deleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyDeployment(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.Deployment {
		obj, err := f.KubeClient.AppsV1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	},
		time.Minute*2,
		time.Second*5,
	)
}

func (f *Invocation) WaitUntilDeploymentReadyWithSidecar(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.AppsV1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
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

func (f *Invocation) WaitUntilDeploymentReadyWithInitContainer(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.AppsV1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
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

func (f *Invocation) DeployDeployment(name string, replica int32) (*apps.Deployment, error) {
	// Create PVC for Deployment
	pvc, err := f.CreateNewPVC(name)
	if err != nil {
		return &apps.Deployment{}, err
	}
	// Generate Deployment definition
	deployment := f.Deployment(pvc.Name)
	deployment.Name = name
	deployment.Spec.Replicas = &replica

	By("Deploying Deployment: " + deployment.Name)
	createdDeployment, err := f.CreateDeployment(deployment)
	if err != nil {
		return createdDeployment, err
	}
	f.AppendToCleanupList(createdDeployment)

	By("Waiting for Deployment to be ready")
	err = apps_util.WaitUntilDeploymentReady(f.KubeClient, createdDeployment.ObjectMeta)
	Expect(err).NotTo(HaveOccurred())
	// check that we can execute command to the pod.
	// this is necessary because we will exec into the pods and create sample data
	f.EventuallyPodAccessible(createdDeployment.ObjectMeta).Should(BeTrue())

	return createdDeployment, err
}
