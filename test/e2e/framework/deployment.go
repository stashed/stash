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

package framework

import (
	"context"
	"fmt"

	"stash.appscode.dev/apimachinery/apis"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gomodules.xyz/pointer"
	"gomodules.xyz/x/crypto/rand"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
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
			Replicas: pointer.Int32P(1),
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

func (f *Framework) EventuallyDeployment(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.Deployment {
		obj, err := f.KubeClient.AppsV1().Deployments(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	},
		WaitTimeOut,
		PullInterval,
	)
}

func (fi *Invocation) WaitUntilDeploymentReadyWithSidecar(meta metav1.ObjectMeta) error {
	return wait.PollUntilContextTimeout(context.Background(), kutil.RetryInterval, kutil.ReadinessTimeout, true, func(ctx context.Context) (bool, error) {
		if obj, err := fi.KubeClient.AppsV1().Deployments(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.Replicas == obj.Status.ReadyReplicas {
				pods, err := fi.GetAllPods(obj.ObjectMeta)
				if err != nil {
					return false, err
				}
				if len(pods) == 0 {
					return false, nil
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

func (fi *Invocation) DeployDeployment(name string, replica int32, volName string, transformFuncs ...func(dp *apps.Deployment)) (*apps.Deployment, error) {
	// append test case specific suffix so that name does not conflict during parallel test
	pvcName := fmt.Sprintf("%s-%s", volName, fi.app)

	// Generate Deployment definition
	deployment := fi.Deployment(name, pvcName, volName)
	deployment.Spec.Replicas = &replica

	// transformFuncs provides a array of functions that made test specific change on the Deployment
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(&deployment)
	}

	// If the PVC does not exist, create PVC for Deployment
	_, err := fi.CreateNewPVC(pvcName, func(p *core.PersistentVolumeClaim) {
		p.Namespace = deployment.Namespace
	})
	if err != nil && !kerr.IsAlreadyExists(err) {
		return nil, err
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
