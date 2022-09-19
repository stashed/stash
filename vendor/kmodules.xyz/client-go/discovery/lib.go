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

package discovery

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"kmodules.xyz/apiversion"
)

func GetVersion(client discovery.DiscoveryInterface) (string, error) {
	info, err := client.ServerVersion()
	if err != nil {
		return "", err
	}
	gv, err := semver.NewVersion(info.GitVersion)
	if err != nil {
		return "", err
	}
	v := *gv
	v, _ = v.SetPrerelease("")
	v, _ = v.SetMetadata("")
	return v.Original(), nil
}

func GetVersionInfo(client discovery.DiscoveryInterface) (uint64, uint64, uint64, string, string, error) {
	info, err := client.ServerVersion()
	if err != nil {
		return 0, 0, 0, "", "", err
	}
	gv, err := semver.NewVersion(info.GitVersion)
	if err != nil {
		return 0, 0, 0, "", "", err
	}
	return gv.Major(), gv.Minor(), gv.Patch(), gv.Prerelease(), gv.Metadata(), nil
}

func GetBaseVersion(client discovery.DiscoveryInterface) (string, error) {
	info, err := client.ServerVersion()
	if err != nil {
		return "", err
	}
	gv, err := semver.NewVersion(info.GitVersion)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d.%d.0", originalVPrefix(gv), gv.Major(), gv.Minor()), nil
}

func originalVPrefix(v *semver.Version) string {
	// Note, only lowercase v is supported as a prefix by the parser.
	if v.Original() != "" && v.Original()[:1] == "v" {
		return v.Original()[:1]
	}
	return ""
}

func CheckAPIVersion(client discovery.DiscoveryInterface, constraint string) (bool, error) {
	info, err := client.ServerVersion()
	if err != nil {
		return false, err
	}
	cond, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, err
	}
	vPtr, err := semver.NewVersion(info.GitVersion)
	if err != nil {
		return false, err
	}
	v := *vPtr
	v, _ = v.SetPrerelease("")
	v, _ = v.SetMetadata("")
	return cond.Check(&v), nil
}

func IsPreferredAPIResource(client discovery.DiscoveryInterface, groupVersion, kind string) bool {
	if resourceList, err := client.ServerPreferredResources(); discovery.IsGroupDiscoveryFailedError(err) || err == nil {
		for _, resources := range resourceList {
			if resources.GroupVersion != groupVersion {
				continue
			}
			for _, resource := range resources.APIResources {
				if resource.Kind == kind && !strings.ContainsRune(resource.Name, '/') {
					return true
				}
			}
		}
	}
	return false
}

func ExistsGroupVersionKind(client discovery.DiscoveryInterface, groupVersion, kind string) bool {
	if resourceList, err := client.ServerPreferredResources(); discovery.IsGroupDiscoveryFailedError(err) || err == nil {
		for _, resources := range resourceList {
			if resources.GroupVersion != groupVersion {
				continue
			}
			for _, resource := range resources.APIResources {
				if resource.Kind == kind && !strings.ContainsRune(resource.Name, '/') {
					return true
				}
			}
		}
	}
	return false
}

func ExistsGroupKind(client discovery.DiscoveryInterface, group, kind string) bool {
	if resourceList, err := client.ServerPreferredResources(); discovery.IsGroupDiscoveryFailedError(err) || err == nil {
		for _, resources := range resourceList {
			gv, err := schema.ParseGroupVersion(resources.GroupVersion)
			if err != nil {
				return false
			}
			if gv.Group != group {
				continue
			}
			for _, resource := range resources.APIResources {
				if resource.Kind == kind && !strings.ContainsRune(resource.Name, '/') {
					return true
				}
			}
		}
	}
	return false
}

func ExistsGroupKinds(client discovery.DiscoveryInterface, gk schema.GroupKind, otherGks ...schema.GroupKind) bool {
	desired := make(map[schema.GroupKind]bool, 1+len(otherGks))
	desired[gk] = false
	for _, other := range otherGks {
		desired[other] = false
	}

	if resourceList, err := client.ServerPreferredResources(); discovery.IsGroupDiscoveryFailedError(err) || err == nil {
		for _, resources := range resourceList {
			gv, err := schema.ParseGroupVersion(resources.GroupVersion)
			if err != nil {
				return false
			}
			for _, resource := range resources.APIResources {
				x := schema.GroupKind{
					Group: gv.Group,
					Kind:  resource.Kind,
				}
				if _, found := desired[x]; found {
					desired[x] = true
				}
			}
		}
	}

	for _, found := range desired {
		if !found {
			return false
		}
	}
	return true
}

type KnownBug struct {
	URL string
	Fix string
}

func (e *KnownBug) Error() string {
	return "Bug: " + e.URL + ". To fix, " + e.Fix
}

var (
	err62649_K1_9  = &KnownBug{URL: "https://github.com/kubernetes/kubernetes/pull/62649", Fix: "upgrade to Kubernetes 1.9.8 or later."}
	err62649_K1_10 = &KnownBug{URL: "https://github.com/kubernetes/kubernetes/pull/62649", Fix: "upgrade to Kubernetes 1.10.2 or later."}
	err83778_K1_16 = &KnownBug{URL: "https://github.com/kubernetes/kubernetes/pull/83787", Fix: "upgrade to Kubernetes 1.16.2 or later."}
)

var (
	DefaultConstraint          = ">= 1.11.0"
	DefaultBlackListedVersions = map[string]error{
		"1.16.0": err83778_K1_16,
		"1.16.1": err83778_K1_16,
	}
	DefaultBlackListedMultiMasterVersions = map[string]error{
		"1.9.0":  err62649_K1_9,
		"1.9.1":  err62649_K1_9,
		"1.9.2":  err62649_K1_9,
		"1.9.3":  err62649_K1_9,
		"1.9.4":  err62649_K1_9,
		"1.9.5":  err62649_K1_9,
		"1.9.6":  err62649_K1_9,
		"1.9.7":  err62649_K1_9,
		"1.10.0": err62649_K1_10,
		"1.10.1": err62649_K1_10,
	}
)

func IsDefaultSupportedVersion(kc kubernetes.Interface) error {
	return IsSupportedVersion(
		kc,
		DefaultConstraint,
		DefaultBlackListedVersions,
		DefaultBlackListedMultiMasterVersions)
}

func IsSupportedVersion(kc kubernetes.Interface, constraint string, blackListedVersions map[string]error, blackListedMultiMasterVersions map[string]error) error {
	info, err := kc.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	klog.Infof("Kubernetes version: %#v\n", info)

	gv, err := semver.NewVersion(info.GitVersion)
	if err != nil {
		return err
	}
	v := *gv
	v, _ = v.SetPrerelease("")
	v, _ = v.SetMetadata("")

	nodes, err := kc.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/master",
	})
	if err != nil {
		return err
	}
	multiMaster := len(nodes.Items) > 1

	return checkVersion(&v, multiMaster, constraint, blackListedVersions, blackListedMultiMasterVersions)
}

func checkVersion(v *semver.Version, multiMaster bool, constraint string, blackListedVersions map[string]error, blackListedMultiMasterVersions map[string]error) error {
	vs := v.String()

	if constraint != "" {
		c, err := semver.NewConstraint(constraint)
		if err != nil {
			return err
		}
		if !c.Check(v) {
			return fmt.Errorf("kubernetes version %s fails constraint %s", vs, constraint)
		}
	}

	if e, ok := blackListedVersions[v.Original()]; ok {
		return errors.Wrapf(e, "kubernetes version %s is blacklisted", v.Original())
	}
	if e, ok := blackListedVersions[vs]; ok {
		return errors.Wrapf(e, "kubernetes version %s is blacklisted", vs)
	}

	if multiMaster {
		if e, ok := blackListedMultiMasterVersions[v.Original()]; ok {
			return errors.Wrapf(e, "kubernetes version %s is blacklisted for multi-master cluster", v.Original())
		}
		if e, ok := blackListedMultiMasterVersions[vs]; ok {
			return errors.Wrapf(e, "kubernetes version %s is blacklisted for multi-master cluster", vs)
		}
	}
	return nil
}

func HasGVK(client discovery.DiscoveryInterface, groupVersion, kind string) (bool, error) {
	_, resourceList, err := client.ServerGroupsAndResources()
	if discovery.IsGroupDiscoveryFailedError(err) || err == nil {
		for _, resources := range resourceList {
			if resources.GroupVersion != groupVersion {
				continue
			}
			for _, resource := range resources.APIResources {
				if resource.Kind == kind && !strings.ContainsRune(resource.Name, '/') {
					return true, nil
				}
			}
		}
	}
	return false, err
}

func ListAPIVersions(c discovery.DiscoveryInterface, group, kind string) ([]string, error) {
	_, resourceList, err := c.ServerGroupsAndResources()

	out := make([]string, 0)
	if discovery.IsGroupDiscoveryFailedError(err) || err == nil {
		for _, resources := range resourceList {
			gv, err := schema.ParseGroupVersion(resources.GroupVersion)
			if err != nil {
				return nil, err
			}
			if gv.Group != group {
				continue
			}
			for _, resource := range resources.APIResources {
				if resource.Kind == kind && !strings.ContainsRune(resource.Name, '/') {
					out = append(out, gv.Version)
				}
			}
		}
	}
	if len(out) > 1 {
		sort.Slice(out, func(i, j int) bool {
			return apiversion.MustCompare(out[i], out[j]) > 0
		})
	}
	return out, nil
}
