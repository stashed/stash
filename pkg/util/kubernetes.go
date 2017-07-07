package util

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/appscode/go/types"
	rapi "github.com/appscode/stash/api"
	sapi "github.com/appscode/stash/api"
	scs "github.com/appscode/stash/client/clientset"
	"github.com/appscode/stash/pkg/docker"
	"github.com/cenkalti/backoff"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	StashContainer = "stash"

	LocalVolumeName      = "stash-local"
	ScratchDirVolumeName = "stash-scratchdir"
	PodinfoVolumeName    = "stash-podinfo"
)

func IsPreferredAPIResource(kubeClient clientset.Interface, groupVersion, kind string) bool {
	if resourceList, err := kubeClient.Discovery().ServerPreferredResources(); err == nil {
		for _, resources := range resourceList {
			if resources.GroupVersion != groupVersion {
				continue
			}
			for _, resource := range resources.APIResources {
				if resources.GroupVersion == groupVersion && resource.Kind == kind {
					return true
				}
			}
		}
	}
	return false
}

func FindRestic(stashClient scs.ExtensionInterface, obj metav1.ObjectMeta) (*sapi.Restic, error) {
	restics, err := stashClient.Restics(obj.Namespace).List(metav1.ListOptions{LabelSelector: labels.Everything().String()})
	if kerr.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	result := make([]*sapi.Restic, 0)
	for _, restic := range restics.Items {
		if selector, err := metav1.LabelSelectorAsSelector(&restic.Spec.Selector); err == nil {
			if selector.Matches(labels.Set(obj.Labels)) {
				result = append(result, &restic)
			}
		}
	}
	if len(result) > 1 {
		var msg bytes.Buffer
		msg.WriteString(fmt.Sprintf("Workload %s@%s matches multiple Restics:", obj.Name, obj.Namespace))
		for i, restic := range result {
			if i > 0 {
				msg.WriteString(", ")
			}
			msg.WriteString(restic.Name)
		}
		return nil, errors.New(msg.String())
	} else if len(result) == 1 {
		return result[0], nil
	}
	return nil, nil
}

func WaitUntilReplicaSetReady(kubeClient clientset.Interface, meta metav1.ObjectMeta) error {
	return backoff.Retry(func() error {
		if obj, err := kubeClient.ExtensionsV1beta1().ReplicaSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if types.Int32(obj.Spec.Replicas) == obj.Status.ReadyReplicas {
				return nil
			}
		}
		return errors.New("check again")
	}, backoff.NewConstantBackOff(2*time.Second))
}

func WaitUntilDeploymentExtensionReady(kubeClient clientset.Interface, meta metav1.ObjectMeta) error {
	return backoff.Retry(func() error {
		if obj, err := kubeClient.ExtensionsV1beta1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if types.Int32(obj.Spec.Replicas) == obj.Status.ReadyReplicas {
				return nil
			}
		}
		return errors.New("check again")
	}, backoff.NewConstantBackOff(2*time.Second))
}

func WaitUntilDeploymentAppReady(kubeClient clientset.Interface, meta metav1.ObjectMeta) error {
	return backoff.Retry(func() error {
		if obj, err := kubeClient.AppsV1beta1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if types.Int32(obj.Spec.Replicas) == obj.Status.ReadyReplicas {
				return nil
			}
		}
		return errors.New("check again")
	}, backoff.NewConstantBackOff(2*time.Second))
}

func WaitUntilDaemonSetReady(kubeClient clientset.Interface, meta metav1.ObjectMeta) error {
	return backoff.Retry(func() error {
		if obj, err := kubeClient.ExtensionsV1beta1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.DesiredNumberScheduled == obj.Status.NumberReady {
				return nil
			}
		}
		return errors.New("check again")
	}, backoff.NewConstantBackOff(2*time.Second))
}

func WaitUntilReplicationControllerReady(kubeClient clientset.Interface, meta metav1.ObjectMeta) error {
	return backoff.Retry(func() error {
		if obj, err := kubeClient.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if types.Int32(obj.Spec.Replicas) == obj.Status.ReadyReplicas {
				return nil
			}
		}
		return errors.New("check again")
	}, backoff.NewConstantBackOff(2*time.Second))
}

func WaitUntilSidecarAdded(kubeClient clientset.Interface, namespace string, selector *metav1.LabelSelector) error {
	return backoff.Retry(func() error {
		r, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return err
		}
		pods, err := kubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: r.String()})
		if err != nil {
			return err
		}

		var podsToRestart []apiv1.Pod
		for _, pod := range pods.Items {
			found := false
			for _, c := range pod.Spec.Containers {
				if c.Name == StashContainer {
					found = true
					break
				}
			}
			if !found {
				podsToRestart = append(podsToRestart, pod)
			}
		}
		if len(podsToRestart) == 0 {
			return nil
		}
		for _, pod := range podsToRestart {
			kubeClient.CoreV1().Pods(namespace).Delete(pod.Name, &metav1.DeleteOptions{})
		}
		return errors.New("check again")
	}, backoff.NewConstantBackOff(3*time.Second))
}

func WaitUntilSidecarRemoved(kubeClient clientset.Interface, namespace string, selector *metav1.LabelSelector) error {
	return backoff.Retry(func() error {
		r, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return err
		}
		pods, err := kubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: r.String()})
		if err != nil {
			return err
		}

		var podsToRestart []apiv1.Pod
		for _, pod := range pods.Items {
			found := false
			for _, c := range pod.Spec.Containers {
				if c.Name == StashContainer {
					found = true
					break
				}
			}
			if found {
				podsToRestart = append(podsToRestart, pod)
			}
		}
		if len(podsToRestart) == 0 {
			return nil
		}
		for _, pod := range podsToRestart {
			kubeClient.CoreV1().Pods(namespace).Delete(pod.Name, &metav1.DeleteOptions{})
		}
		return errors.New("check again")
	}, backoff.NewConstantBackOff(3*time.Second))
}

func RestartPods(kubeClient clientset.Interface, namespace string, selector *metav1.LabelSelector) error {
	r, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return err
	}
	return kubeClient.CoreV1().Pods(namespace).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: r.String(),
	})
}

func GetString(m map[string]string, key string) string {
	if m == nil {
		return ""
	}
	return m[key]
}

func CreateSidecarContainer(r *rapi.Restic, tag, workload string) apiv1.Container {
	if r.Annotations != nil {
		if v, ok := r.Annotations[sapi.VersionTag]; ok {
			tag = v
		}
	}
	sidecar := apiv1.Container{
		Name:            StashContainer,
		Image:           docker.ImageOperator + ":" + tag,
		ImagePullPolicy: apiv1.PullIfNotPresent,
		Args: []string{
			"schedule",
			"--restic-name=" + r.Name,
			"--workload=" + workload,
		},
		Env: []apiv1.EnvVar{
			{
				Name: "NODE_NAME",
				ValueFrom: &apiv1.EnvVarSource{
					FieldRef: &apiv1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			{
				Name: "POD_NAME",
				ValueFrom: &apiv1.EnvVarSource{
					FieldRef: &apiv1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
		Resources: r.Spec.Resources,
		VolumeMounts: []apiv1.VolumeMount{
			{
				Name:      ScratchDirVolumeName,
				MountPath: "/tmp",
			},
			{
				Name:      PodinfoVolumeName,
				MountPath: "/etc/stash",
			},
		},
	}
	if tag == "canary" {
		sidecar.ImagePullPolicy = apiv1.PullAlways
		sidecar.Args = append(sidecar.Args, "--v=5")
	} else {
		sidecar.Args = append(sidecar.Args, "--v=3")
	}
	for _, srcVol := range r.Spec.VolumeMounts {
		sidecar.VolumeMounts = append(sidecar.VolumeMounts, apiv1.VolumeMount{
			Name:      srcVol.Name,
			MountPath: srcVol.MountPath,
			SubPath:   srcVol.SubPath,
			ReadOnly:  true,
		})
	}
	if r.Spec.Backend.Local != nil {
		sidecar.VolumeMounts = append(sidecar.VolumeMounts, apiv1.VolumeMount{
			Name:      LocalVolumeName,
			MountPath: r.Spec.Backend.Local.Path,
		})
	}
	return sidecar
}

func EnsureContainerDeleted(containers []apiv1.Container, name string) []apiv1.Container {
	for i, c := range containers {
		if c.Name == name {
			return append(containers[:i], containers[i+1:]...)
		}
	}
	return containers
}

func UpsertContainer(containers []apiv1.Container, nv apiv1.Container) []apiv1.Container {
	for i, vol := range containers {
		if vol.Name == nv.Name {
			containers[i] = nv
			return containers
		}
	}
	return append(containers, nv)
}

func UpsertVolume(volumes []apiv1.Volume, nv apiv1.Volume) []apiv1.Volume {
	for i, vol := range volumes {
		if vol.Name == nv.Name {
			volumes[i] = nv
			return volumes
		}
	}
	return append(volumes, nv)
}

func UpsertScratchVolume(volumes []apiv1.Volume) []apiv1.Volume {
	return UpsertVolume(volumes, apiv1.Volume{
		Name: ScratchDirVolumeName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	})
}

// https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#store-pod-fields
func UpsertDownwardVolume(volumes []apiv1.Volume) []apiv1.Volume {
	return UpsertVolume(volumes, apiv1.Volume{
		Name: PodinfoVolumeName,
		VolumeSource: apiv1.VolumeSource{
			DownwardAPI: &apiv1.DownwardAPIVolumeSource{
				Items: []apiv1.DownwardAPIVolumeFile{
					{
						Path: "labels",
						FieldRef: &apiv1.ObjectFieldSelector{
							FieldPath: "metadata.labels",
						},
					},
				},
			},
		},
	})
}

func MergeLocalVolume(volumes []apiv1.Volume, old, new *sapi.Restic) []apiv1.Volume {
	oldPos := -1
	if old != nil && old.Spec.Backend.Local != nil {
		for i, vol := range volumes {
			if vol.Name == LocalVolumeName {
				oldPos = i
				break
			}
		}
	}
	if new.Spec.Backend.Local != nil {
		if oldPos != -1 {
			volumes[oldPos] = apiv1.Volume{Name: LocalVolumeName, VolumeSource: new.Spec.Backend.Local.VolumeSource}
		} else {
			volumes = UpsertVolume(volumes, apiv1.Volume{Name: LocalVolumeName, VolumeSource: new.Spec.Backend.Local.VolumeSource})
		}
	} else {
		if oldPos != -1 {
			volumes = append(volumes[:oldPos], volumes[oldPos+1:]...)
		}
	}
	return volumes
}

func EnsureVolumeDeleted(volumes []apiv1.Volume, name string) []apiv1.Volume {
	for i, v := range volumes {
		if v.Name == name {
			return append(volumes[:i], volumes[i+1:]...)
		}
	}
	return volumes
}

func ResticEqual(old, new *sapi.Restic) bool {
	var oldSpec, newSpec *sapi.ResticSpec
	if old != nil {
		oldSpec = &old.Spec
	}
	if new != nil {
		newSpec = &new.Spec
	}
	return reflect.DeepEqual(oldSpec, newSpec)
}
