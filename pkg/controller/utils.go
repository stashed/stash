package controller

import (
	rapi "github.com/appscode/stash/api"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/docker"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (c *Controller) IsPreferredAPIResource(groupVersion, kind string) bool {
	if resourceList, err := c.KubeClient.Discovery().ServerPreferredResources(); err == nil {
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

func (c *Controller) FindRestic(obj metav1.ObjectMeta) (*sapi.Restic, error) {
	restics, err := c.StashClient.Restics(obj.Namespace).List(metav1.ListOptions{LabelSelector: labels.Everything().String()})
	if kerr.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	for _, restic := range restics.Items {
		selector, err := metav1.LabelSelectorAsSelector(&restic.Spec.Selector)
		//return nil, fmt.Errorf("invalid selector: %v", err)
		if err == nil {
			if selector.Matches(labels.Set(obj.Labels)) {
				return &restic, nil
			}
		}
	}
	return nil, nil
}

func (c *Controller) restartPods(namespace string, selector *metav1.LabelSelector) error {
	r, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return err
	}
	return c.KubeClient.CoreV1().Pods(namespace).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: r.String(),
	})
}

func getString(m map[string]string, key string) string {
	if m == nil {
		return ""
	}
	return m[key]
}

func (c *Controller) GetSidecarContainer(r *rapi.Restic, workload string, prefixHostname bool) apiv1.Container {
	tag := c.SidecarImageTag
	if r.Annotations != nil {
		if v, ok := r.Annotations[sapi.VersionTag]; ok {
			tag = v
		}
	}

	sidecar := apiv1.Container{
		Name:            docker.StashContainer,
		Image:           docker.ImageOperator + ":" + tag,
		ImagePullPolicy: apiv1.PullIfNotPresent,
		Args: []string{
			"schedule",
			"--v=3",
			"--workload=" + workload,
			"--namespace=" + r.Namespace,
			"--name=" + r.Name,
		},
		VolumeMounts: []apiv1.VolumeMount{
			{
				Name:      ScratchDirVolumeName,
				MountPath: "/tmp",
			},
		},
	}
	if prefixHostname {
		sidecar.Args = append(sidecar.Args, "--prefixHostname=true")
	}
	if r.Spec.Backend.Local != nil {
		sidecar.VolumeMounts = append(sidecar.VolumeMounts, apiv1.VolumeMount{
			Name:      r.Spec.Backend.Local.Volume.Name,
			MountPath: r.Spec.Backend.Local.Path,
		})
	}
	return sidecar
}

func (c *Controller) addAnnotation(r *rapi.Restic) {
	if r.ObjectMeta.Annotations == nil {
		r.ObjectMeta.Annotations = make(map[string]string)
	}
	r.ObjectMeta.Annotations[sapi.VersionTag] = c.SidecarImageTag
}

func removeContainer(c []apiv1.Container, name string) []apiv1.Container {
	for i, v := range c {
		if v.Name == name {
			c = append(c[:i], c[i+1:]...)
			break
		}
	}
	return c
}

func addScratchVolume(volumes []apiv1.Volume) []apiv1.Volume {
	return append(volumes, apiv1.Volume{
		Name: ScratchDirVolumeName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	})
}

// https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#store-pod-fields
func addDownwardVolume(volumes []apiv1.Volume) []apiv1.Volume {
	return append(volumes, apiv1.Volume{
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

func removeVolume(volumes []apiv1.Volume, name string) []apiv1.Volume {
	for i, v := range volumes {
		if v.Name == name {
			volumes = append(volumes[:i], volumes[i+1:]...)
			break
		}
	}
	return volumes
}
