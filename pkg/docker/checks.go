package docker

import (
	"github.com/heroku/docker-registry-client/registry"
	core "k8s.io/api/core/v1"
)

const (
	registryUrl  = "https://registry-1.docker.io/"
	ACRegistry   = "appscode"
	ImageStash   = "stash"
	ImageKubectl = "kubectl"
)

type Docker struct {
	Registry, Image, Tag string
}

func (docker Docker) Verify(secrets []core.LocalObjectReference) error {
	if docker.Registry == ACRegistry {
		repository := docker.Registry + "/" + docker.Image
		if hub, err := registry.New(registryUrl, "", ""); err != nil {
			return err
		} else {
			_, err = hub.Manifest(repository, docker.Tag)
			return err
		}
	} else { // TODO @ Dipta: verify private repository
		return nil
	}
}

func (docker Docker) ToContainerImage() string {
	return docker.Registry + "/" + docker.Image + ":" + docker.Tag
}
