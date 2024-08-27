/*
Copyright AppsCode Inc. and Contributors

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

package apiutil

import (
	"context"
	"strings"

	kmapi "kmodules.xyz/client-go/api/v1"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CollectImageInfo(kc client.Client, pod *core.Pod, images map[string]kmapi.ImageInfo, fullLineage bool) (map[string]kmapi.ImageInfo, error) {
	var lineage []kmapi.ObjectInfo

	var err error
	if fullLineage {
		lineage, err = DetectLineage(context.TODO(), kc, pod)
		if err != nil {
			return images, err
		}
	} else {
		lineage = []kmapi.ObjectInfo{
			{
				Resource: kmapi.ResourceID{
					Group:   "",
					Version: "v1",
					Name:    "pods",
					Kind:    "Pod",
					Scope:   kmapi.NamespaceScoped,
				},
				Ref: kmapi.ObjectReference{
					Namespace: pod.Namespace,
					Name:      pod.Name,
				},
			},
		}
	}

	refs := map[string][]string{}
	for _, c := range pod.Spec.Containers {
		si, sid := findContainerStatus(c.Name, pod.Status.ContainerStatuses)
		ref, err := GetImageRef(c.Image, si, sid)
		if err != nil {
			return images, err
		}
		refs[ref] = append(refs[ref], c.Name)
	}
	for _, c := range pod.Spec.InitContainers {
		si, sid := findContainerStatus(c.Name, pod.Status.InitContainerStatuses)
		ref, err := GetImageRef(c.Image, si, sid)
		if err != nil {
			return images, err
		}
		refs[ref] = append(refs[ref], c.Name)
	}
	for _, c := range pod.Spec.EphemeralContainers {
		si, sid := findContainerStatus(c.Name, pod.Status.EphemeralContainerStatuses)
		ref, err := GetImageRef(c.Image, si, sid)
		if err != nil {
			return images, err
		}
		refs[ref] = append(refs[ref], c.Name)
	}

	for ref, containers := range refs {
		iu, ok := images[ref]
		if !ok {
			iu = kmapi.ImageInfo{
				Image:    ref,
				Lineages: nil,
				PullCredentials: &kmapi.PullCredentials{
					Namespace:          pod.Namespace,
					SecretRefs:         pod.Spec.ImagePullSecrets,
					ServiceAccountName: pod.Spec.ServiceAccountName,
				},
			}
		}
		iu.Lineages = append(iu.Lineages, kmapi.Lineage{
			Chain:      lineage,
			Containers: containers,
		})
		images[ref] = iu
	}

	return images, nil
}

func CollectPullCredentials(pod *core.Pod, refs map[string]kmapi.PullCredentials) (map[string]kmapi.PullCredentials, error) {
	for _, c := range pod.Spec.Containers {
		si, sid := findContainerStatus(c.Name, pod.Status.ContainerStatuses)
		ref, err := GetImageRef(c.Image, si, sid)
		if err != nil {
			return refs, err
		}
		refs[ref] = kmapi.PullCredentials{
			Namespace:          pod.Namespace,
			SecretRefs:         pod.Spec.ImagePullSecrets,
			ServiceAccountName: pod.Spec.ServiceAccountName,
		}
	}
	for _, c := range pod.Spec.InitContainers {
		si, sid := findContainerStatus(c.Name, pod.Status.InitContainerStatuses)
		ref, err := GetImageRef(c.Image, si, sid)
		if err != nil {
			return refs, err
		}
		refs[ref] = kmapi.PullCredentials{
			Namespace:          pod.Namespace,
			SecretRefs:         pod.Spec.ImagePullSecrets,
			ServiceAccountName: pod.Spec.ServiceAccountName,
		}
	}
	for _, c := range pod.Spec.EphemeralContainers {
		si, sid := findContainerStatus(c.Name, pod.Status.EphemeralContainerStatuses)
		ref, err := GetImageRef(c.Image, si, sid)
		if err != nil {
			return refs, err
		}
		refs[ref] = kmapi.PullCredentials{
			Namespace:          pod.Namespace,
			SecretRefs:         pod.Spec.ImagePullSecrets,
			ServiceAccountName: pod.Spec.ServiceAccountName,
		}
	}

	return refs, nil
}

func GetImageRef(containerImage, statusImage, statusImageID string) (string, error) {
	var img string

	if strings.ContainsRune(containerImage, '@') {
		img = containerImage
	} else if strings.ContainsRune(statusImage, '@') {
		img = statusImage
	} else {
		// take the hash from status.ImageID and add to c.Image
		if strings.Contains(statusImageID, "://") {
			statusImageID = statusImageID[strings.Index(statusImageID, "://")+3:] // remove docker-pullable://
		}

		// Now check imageID is using same repo as the contianerImage
		// This will not be same for images loaded into a KIND cluster

		isSameContext := func(img1, img2 string) bool {
			ref1, err := name.ParseReference(img1)
			if err != nil {
				return false
			}
			ref2, err := name.ParseReference(img2)
			if err != nil {
				return false
			}
			return ref1.Context().String() == ref2.Context().String()
		}

		_, digest, ok := strings.Cut(statusImageID, "@")
		if isSameContext(containerImage, statusImageID) && ok {
			img = containerImage + "@" + digest
		} else {
			img = containerImage
			// return "", fmt.Errorf("missing digest in pod %s container %s imageID %s", pod, status.Name, status.ImageID)
		}
	}

	ref, err := name.ParseReference(img)
	if err != nil {
		return "", errors.Wrapf(err, "ref=%s", img)
	}
	return ref.Name(), nil
}

func findContainerStatus(name string, statuses []core.ContainerStatus) (string, string) {
	for _, s := range statuses {
		if s.Name == name {
			return s.Image, s.ImageID
		}
	}
	return "", ""
}

func DetectLineage(ctx context.Context, kc client.Client, obj client.Object) ([]kmapi.ObjectInfo, error) {
	return findLineage(ctx, kc, obj, nil)
}

func findLineage(ctx context.Context, kc client.Client, obj client.Object, result []kmapi.ObjectInfo) ([]kmapi.ObjectInfo, error) {
	ref := metav1.GetControllerOfNoCopy(obj)
	if ref != nil {
		var owner unstructured.Unstructured
		owner.SetAPIVersion(ref.APIVersion)
		owner.SetKind(ref.Kind)
		if err := kc.Get(ctx, client.ObjectKey{Namespace: obj.GetNamespace(), Name: ref.Name}, &owner); client.IgnoreNotFound(err) != nil {
			return result, err
		} else if err == nil { // ignore not found error, owner might be already deleted
			var err error
			result, err = findLineage(ctx, kc, &owner, result)
			if err != nil {
				return result, err
			}
		}
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	mapping, err := kc.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	result = append(result, kmapi.ObjectInfo{
		Resource: *kmapi.NewResourceID(mapping),
		Ref: kmapi.ObjectReference{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		},
	})
	return result, nil
}
