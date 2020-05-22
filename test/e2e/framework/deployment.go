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
	"time"

	"stash.appscode.dev/apimachinery/apis"

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
	meta_util "kmodules.xyz/client-go/meta"
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

func (fi *Invocation) Deployment(name, pvcName, volName string) apps.Deployment {
	name = rand.WithUniqSuffix(fmt.Sprintf("%s-%s", name, fi.app))
	labels := map[string]string{
		"app":  name,
		"kind": "deployment",
	}
	return apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.DeploymentSpec{
			Replicas: types.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: fi.PodTemplate(labels, pvcName, volName),
		},
	}
}

func (f *Framework) CreateDeployment(obj apps.Deployment) (*apps.Deployment, error) {
	return f.KubeClient.AppsV1().Deployments(obj.Namespace).Create(context.TODO(), &obj, metav1.CreateOptions{})
}

func (f *Framework) DeleteDeployment(meta metav1.ObjectMeta) error {
	err := f.KubeClient.AppsV1().Deployments(meta.Namespace).Delete(context.TODO(), meta.Name, meta_util.DeleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyDeployment(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.Deployment {
		obj, err := f.KubeClient.AppsV1().Deployments(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	},
		time.Minute*2,
		time.Second*5,
	)
}

func (fi *Invocation) WaitUntilDeploymentReadyWithSidecar(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := fi.KubeClient.AppsV1().Deployments(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{}); err == nil {
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

func (fi *Invocation) WaitUntilDeploymentReadyWithInitContainer(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := fi.KubeClient.AppsV1().Deployments(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{}); err == nil {
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

func (fi *Invocation) DeployDeployment(name string, replica int32, volName string, transformFuncs ...func(dp *apps.Deployment)) (*apps.Deployment, error) {
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

	// Generate Deployment definition
	deployment := fi.Deployment(name, pvc.Name, volName)
	deployment.Spec.Replicas = &replica

	// transformFuncs provides a array of functions that made test specific change on the Deployment
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(&deployment)
	}

	By("Deploying Deployment: " + deployment.Name)
	createdDeployment, err := fi.CreateDeployment(deployment)
	if err != nil {
		return createdDeployment, err
	}
	fi.AppendToCleanupList(createdDeployment)

	By("Waiting for Deployment to be ready")
	err = apps_util.WaitUntilDeploymentReady(context.TODO(), fi.KubeClient, createdDeployment.ObjectMeta)
	Expect(err).NotTo(HaveOccurred())
	// check that we can execute command to the pod.
	// this is necessary because we will exec into the pods and create sample data
	fi.EventuallyAllPodsAccessible(createdDeployment.ObjectMeta).Should(BeTrue())

	return createdDeployment, err
}
