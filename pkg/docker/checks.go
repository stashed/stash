package docker

const (
	ACRegistry   = "emruzhossain"
	ImageStash   = "stash"
	ImageKubectl = "kubectl"
)

type Docker struct {
	Registry, Image, Tag string
}

func (docker Docker) ToContainerImage() string {
	return docker.Registry + "/" + docker.Image + ":" + docker.Tag
}
