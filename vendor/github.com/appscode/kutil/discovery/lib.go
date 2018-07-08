package discovery

import (
	"github.com/appscode/go-version"
	"k8s.io/client-go/discovery"
)

func GetVersion(client discovery.DiscoveryInterface) (string, error) {
	info, err := client.ServerVersion()
	if err != nil {
		return "", err
	}
	gv, err := version.NewVersion(info.GitVersion)
	if err != nil {
		return "", err
	}
	return gv.ToMutator().ResetMetadata().ResetPrerelease().String(), nil
}

func GetBaseVersion(client discovery.DiscoveryInterface) (string, error) {
	info, err := client.ServerVersion()
	if err != nil {
		return "", err
	}
	gv, err := version.NewVersion(info.GitVersion)
	if err != nil {
		return "", err
	}
	return gv.ToMutator().ResetMetadata().ResetPrerelease().ResetPatch().String(), nil
}

func CheckAPIVersion(client discovery.DiscoveryInterface, constraint string) (bool, error) {
	info, err := client.ServerVersion()
	if err != nil {
		return false, err
	}
	cond, err := version.NewConstraint(constraint)
	if err != nil {
		return false, err
	}
	v, err := version.NewVersion(info.GitVersion)
	if err != nil {
		return false, err
	}
	return cond.Check(v.ToMutator().ResetPrerelease().ResetMetadata().Done()), nil
}

func IsPreferredAPIResource(client discovery.DiscoveryInterface, groupVersion, kind string) bool {
	if resourceList, err := client.ServerPreferredResources(); err == nil {
		for _, resources := range resourceList {
			if resources.GroupVersion != groupVersion {
				continue
			}
			for _, resource := range resources.APIResources {
				if resource.Kind == kind {
					return true
				}
			}
		}
	}
	return false
}
