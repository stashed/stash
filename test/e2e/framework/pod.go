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
	"bytes"
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gomodules.xyz/x/crypto/rand"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	v1 "kmodules.xyz/client-go/core/v1"
)

const (
	TestSourceDataVolumeName = "source-data"
	TestSourceDataMountPath  = "/source/data"
	TestSafeDataMountPath    = "/safe/data"
	OperatorNamespace        = "kube-system"
	ContainerBusyBox         = "busybox"
)

func (fi *Invocation) PodTemplate(labels map[string]string, pvcName, volName string) core.PodTemplateSpec {
	return core.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            ContainerBusyBox,
					Image:           "busybox",
					ImagePullPolicy: core.PullIfNotPresent,
					Command: []string{
						"sleep",
						"3600",
					},
					VolumeMounts: []core.VolumeMount{
						{
							Name:      volName,
							MountPath: TestSourceDataMountPath,
						},
					},
				},
			},
			Volumes: []core.Volume{
				{
					Name: volName,
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}
}

func (f *Framework) GetPod(meta metav1.ObjectMeta) (*core.Pod, error) {
	podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		if bytes.HasPrefix([]byte(pod.Name), []byte(meta.Name)) {
			return &pod, nil
		}
	}
	return nil, fmt.Errorf("no pod found for workload %v", meta.Name)
}

func (f *Framework) GetAllPods(meta metav1.ObjectMeta) ([]core.Pod, error) {
	pods := make([]core.Pod, 0)
	labelSelector := fields.SelectorFromSet(meta.Labels)
	podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector.String()})
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, meta.Name) {
			pods = append(pods, pod)
		}
	}
	return pods, nil
}

func (f *Framework) GetOperatorPod() (*core.Pod, error) {
	podList, err := f.KubeClient.CoreV1().Pods(OperatorNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, "stash") {
			for _, c := range pod.Spec.Containers {
				if c.Name == "operator" {
					return &pod, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("operator pod not found")
}

func (f *Framework) GetMinioPod() (*core.Pod, error) {
	podList, err := f.KubeClient.CoreV1().Pods(f.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, MinioServer) {
			return &pod, nil
		}
	}
	return nil, fmt.Errorf("operator pod not found")
}

func (fi *Invocation) Pod(pvcName string) core.Pod {
	podName := rand.WithUniqSuffix(fmt.Sprintf("test-write-source-%s", fi.app))
	labels := map[string]string{
		"app": podName,
	}
	return core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            ContainerBusyBox,
					Image:           "busybox",
					ImagePullPolicy: core.PullIfNotPresent,
					Command: []string{
						"/bin/sh",
						"-c",
					},
					Args: []string{
						"set -x; while true; do sleep 30; done;",
					},
					VolumeMounts: []core.VolumeMount{
						{
							Name:      TestSourceDataVolumeName,
							MountPath: TestSourceDataMountPath,
						},
					},
				},
			},
			Volumes: []core.Volume{
				{
					Name: TestSourceDataVolumeName,
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}
}

func (fi *Invocation) CreatePod(pod core.Pod) (*core.Pod, error) {
	return fi.KubeClient.CoreV1().Pods(pod.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
}

func (fi *Invocation) DeletePod(meta metav1.ObjectMeta) error {
	err := fi.KubeClient.CoreV1().Pods(meta.Namespace).Delete(context.TODO(), meta.Name, metav1.DeleteOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyAllPodsAccessible(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() bool {
		labelSelector := fields.SelectorFromSet(meta.Labels)
		podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector.String()})
		Expect(err).NotTo(HaveOccurred())

		if len(podList.Items) == 0 {
			return false
		}

		allPodAccessible := true
		for _, pod := range podList.Items {
			_, err := f.ExecOnPod(&pod, "ls", "-R")
			if err != nil {
				allPodAccessible = false
				break
			}
		}
		return allPodAccessible
	},
		WaitTimeOut,
		PullInterval,
	)
}

func (f *Framework) EventuallyPodAccessible(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() bool {
		pod, err := f.KubeClient.CoreV1().Pods(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		_, err = f.ExecOnPod(pod, "ls", "-R")
		return err == nil
	}, WaitTimeOut, PullInterval)
}

func (fi *Invocation) DeployPod(pvcName string) (*core.Pod, error) {
	// Generate Pod definition
	pod := fi.Pod(pvcName)

	By(fmt.Sprintf("Deploying Pod: %s/%s", pod.Namespace, pod.Name))
	createdPod, err := fi.CreatePod(pod)
	if err != nil {
		return createdPod, err
	}
	fi.AppendToCleanupList(createdPod)

	By("Waiting for Pod to be ready")
	err = v1.WaitUntilPodRunning(context.TODO(), fi.KubeClient, createdPod.ObjectMeta)
	// check that we can execute command to the pod.
	// this is necessary because we will exec into the pods and create sample data
	fi.EventuallyPodAccessible(createdPod.ObjectMeta).Should(BeTrue())

	return createdPod, err
}
