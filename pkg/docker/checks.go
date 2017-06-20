package docker

import (
	docker "github.com/heroku/docker-registry-client/registry"
)

const (
	registryUrl = "https://registry-1.docker.io/"
)

func CheckDockerImageVersion(repository, reference string) error {
	hub, err := docker.New(registryUrl, "", "")
	if err != nil {
		return err
	}

	_, err = hub.Manifest(repository, reference)
	return err
}
