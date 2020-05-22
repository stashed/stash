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

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
	apps_util "kmodules.xyz/client-go/apps/v1"
	meta_util "kmodules.xyz/client-go/meta"
)

func (fi *Invocation) StatefulSet(name, pvcName, volName string) apps.StatefulSet {
	name = rand.WithUniqSuffix(fmt.Sprintf("%s-%s", name, fi.app))
	labels := map[string]string{
		"app":  name,
		"kind": "statefulset",
	}
	return apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas:    types.Int32P(1),
			Template:    fi.PodTemplate(labels, pvcName, volName),
			ServiceName: name,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
		},
	}
}

func (fi *Invocation) StatefulSetForV1beta1API(name, volName string, replica int32) apps.StatefulSet {
	labels := map[string]string{
		"app":  name,
		"kind": "statefulset",
	}
	return apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas:    &replica,
			ServiceName: name,
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
									Name:      volName,
									MountPath: TestSourceDataMountPath,
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []core.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: volName,
					},
					Spec: core.PersistentVolumeClaimSpec{
						AccessModes: []core.PersistentVolumeAccessMode{
							core.ReadWriteOnce,
						},
						StorageClassName: types.StringP(fi.StorageClass),
						Resources: core.ResourceRequirements{
							Requests: core.ResourceList{
								core.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		},
	}
}

func (f *Framework) CreateStatefulSet(obj apps.StatefulSet) (*apps.StatefulSet, error) {
	return f.KubeClient.AppsV1().StatefulSets(obj.Namespace).Create(context.TODO(), &obj, metav1.CreateOptions{})
}

func (f *Framework) DeleteStatefulSet(meta metav1.ObjectMeta) error {
	err := f.KubeClient.AppsV1().StatefulSets(meta.Namespace).Delete(context.TODO(), meta.Name, meta_util.DeleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) EventuallyStatefulSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.StatefulSet {
		obj, err := f.KubeClient.AppsV1().StatefulSets(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

func (fi *Invocation) WaitUntilStatefulSetReadyWithSidecar(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := fi.KubeClient.AppsV1().StatefulSets(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{}); err == nil {
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

func (fi *Invocation) WaitUntilStatefulSetWithInitContainer(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := fi.KubeClient.AppsV1().StatefulSets(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{}); err == nil {
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

func (fi *Invocation) DeployStatefulSet(name string, replica int32, volName string, transformFuncs ...func(ss *apps.StatefulSet)) (*apps.StatefulSet, error) {
	// Generate StatefulSet definition
	name = fmt.Sprintf("%s-%s", name, fi.app)
	ss := fi.StatefulSetForV1beta1API(name, volName, replica)

	// transformFuncs provides a array of functions that made test specific change on the StatefulSet
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(&ss)
	}

	By("Deploying StatefulSet: " + ss.Name)
	createdss, err := fi.CreateStatefulSet(ss)
	if err != nil {
		return createdss, err
	}
	fi.AppendToCleanupList(createdss)

	By("Waiting for StatefulSet to be ready")
	err = apps_util.WaitUntilStatefulSetReady(context.TODO(), fi.KubeClient, createdss.ObjectMeta)
	Expect(err).NotTo(HaveOccurred())
	// check that we can execute command to the pod.
	// this is necessary because we will exec into the pods and create sample data
	fi.EventuallyAllPodsAccessible(createdss.ObjectMeta).Should(BeTrue())

	return createdss, err
}

func (fi *Invocation) DeployStatefulSetWithProbeClient(name string) (*apps.StatefulSet, error) {
	name = fmt.Sprintf("%s-%s", name, fi.app)
	svc, err := fi.CreateService(fi.HeadlessService(name))
	if err != nil {
		return nil, err
	}
	fi.AppendToCleanupList(svc)

	labels := map[string]string{
		"app":  name,
		"kind": "statefulset",
	}
	// Generate StatefulSet definition
	statefulset := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: fi.namespace,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: types.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: name,
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  ProberDemoPodPrefix,
							Image: "appscodeci/prober-demo",
							Args: []string{
								"run-client",
							},
							Env: []core.EnvVar{
								{
									Name:  ExitCodeSuccess,
									Value: "0",
								},
								{
									Name:  ExitCodeFail,
									Value: "1",
								},
							},
							Ports: []core.ContainerPort{
								{
									Name:          HttpPortName,
									ContainerPort: HttpPort,
								},
								{
									Name:          TcpPortName,
									ContainerPort: TcpPort,
								},
							},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      SourceVolume,
									MountPath: TestSourceDataMountPath,
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []core.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: SourceVolume,
					},
					Spec: core.PersistentVolumeClaimSpec{
						AccessModes: []core.PersistentVolumeAccessMode{
							core.ReadWriteOnce,
						},
						StorageClassName: types.StringP(fi.StorageClass),
						Resources: core.ResourceRequirements{
							Requests: core.ResourceList{
								core.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		},
	}

	By("Deploying StatefulSet with Probe Client: " + statefulset.Name)
	createdStatefulSet, err := fi.CreateStatefulSet(*statefulset)
	if err != nil {
		return createdStatefulSet, err
	}
	fi.AppendToCleanupList(createdStatefulSet)

	By("Waiting for StatefulSet to be ready")
	err = apps_util.WaitUntilStatefulSetReady(context.TODO(), fi.KubeClient, createdStatefulSet.ObjectMeta)
	Expect(err).NotTo(HaveOccurred())
	// check that we can execute command to the pod.
	// this is necessary because we will exec into the pods and create sample data
	fi.EventuallyAllPodsAccessible(createdStatefulSet.ObjectMeta).Should(BeTrue())

	return createdStatefulSet, err
}
