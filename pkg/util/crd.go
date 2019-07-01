package util

import (
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/docker"
)

func EnsureUpdateStatusFunction(stashClient cs.Interface, registry, imageTag string) error {
	image := docker.Docker{
		Registry: registry,
		Image:    docker.ImageStash,
		Tag:      imageTag,
	}

	updateStatusFunc := &api_v1beta1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name: "update-status",
		},
		Spec: api_v1beta1.FunctionSpec{
			Image: image.ToContainerImage(),
			Args: []string{
				"update-status",
				"--namespace=${NAMESPACE:=default}",
				"--repository=${REPOSITORY_NAME:=}",
				"--restore-session=${RESTORE_SESSION:=}",
				"--output-dir=${outputDir:=}",
				"--enable-status-subresource=${ENABLE_STATUS_SUBRESOURCE:=false}",
			},
		},
	}

	_, err := stashClient.StashV1beta1().Functions().Create(updateStatusFunc)
	if err != nil && !kerr.IsAlreadyExists(err) {
		return err
	}
	return nil
}
