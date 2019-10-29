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
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
)

func (fi *Invocation) DaemonSet() apps.DaemonSet {
	labels := map[string]string{
		"app":  fi.app,
		"kind": "daemonset",
	}
	daemon := apps.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: core.PodTemplateSpec{
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
								HostPath: &core.HostPathVolumeSource{
									Path: TestSourceDataMountPath,
								},
							},
						},
					},
				},
			},
			UpdateStrategy: apps.DaemonSetUpdateStrategy{
				RollingUpdate: &apps.RollingUpdateDaemonSet{MaxUnavailable: &intstr.IntOrString{IntVal: 1}},
			},
		},
	}
	return daemon
}

func (f *Framework) CreateDaemonSet(obj apps.DaemonSet) (*apps.DaemonSet, error) {
	return f.KubeClient.AppsV1().DaemonSets(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteDaemonSet(meta metav1.ObjectMeta) error {
	err := f.KubeClient.AppsV1().DaemonSets(meta.Namespace).Delete(meta.Name, deleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyDaemonSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.DaemonSet {
		obj, err := f.KubeClient.AppsV1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

func (f *Invocation) WaitUntilDaemonSetReadyWithSidecar(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.AppsV1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.DesiredNumberScheduled == obj.Status.NumberReady {
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

func (f *Invocation) WaitUntilDaemonSetReadyWithInitContainer(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.AppsV1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.DesiredNumberScheduled == obj.Status.NumberReady {
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
