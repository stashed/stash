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
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
	OperatorName             = "stash-operator"
)

func (fi *Invocation) PodTemplate(labels map[string]string, pvcName string) core.PodTemplateSpec {
	return core.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            "busybox",
					Image:           "busybox",
					ImagePullPolicy: core.PullIfNotPresent,
					Command: []string{
						"sleep",
						"3600",
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

func (f *Framework) GetPod(meta metav1.ObjectMeta) (*core.Pod, error) {
	podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{})
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
	podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{LabelSelector: labelSelector.String()})
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, meta.Name) {
			pods = append(pods, pod)
		}
	}
	if len(pods) > 0 {
		return pods, nil
	}
	return nil, fmt.Errorf("no pod found for workload %v", meta.Name)
}

func (f *Framework) GetOperatorPod() (*core.Pod, error) {
	podList, err := f.KubeClient.CoreV1().Pods(OperatorNamespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, OperatorName) {
			return &pod, nil
		}
	}
	return nil, fmt.Errorf("operator pod not found")
}

func (f *Invocation) Pod(pvcName string) core.Pod {
	labels := map[string]string{
		"app": f.app,
	}
	return core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("test-write-source"),
			Namespace: f.namespace,
			Labels:    labels,
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            "busybox",
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

func (f *Invocation) CreatePod(pod core.Pod) (*core.Pod, error) {
	return f.KubeClient.CoreV1().Pods(pod.Namespace).Create(&pod)
}

func (f *Invocation) DeletePod(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().Pods(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyPodAccessible(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() bool {
		labelSelector := fields.SelectorFromSet(meta.Labels)
		podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{LabelSelector: labelSelector.String()})
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
		time.Minute*2,
		time.Second*2,
	)
}

func (f *Invocation) DeployPod(pvcName string) (*core.Pod, error) {
	// Generate Pod definition
	pod := f.Pod(pvcName)

	By(fmt.Sprintf("Deploying Pod: %s/%s", pod.Namespace, pod.Name))
	createdPod, err := f.CreatePod(pod)
	if err != nil {
		return createdPod, err
	}
	f.AppendToCleanupList(createdPod)

	By("Waiting for Pod to be ready")
	err = v1.WaitUntilPodRunning(f.KubeClient, createdPod.ObjectMeta)
	// check that we can execute command to the pod.
	// this is necessary because we will exec into the pods and create sample data
	f.EventuallyPodAccessible(createdPod.ObjectMeta).Should(BeTrue())

	return createdPod, err
}
