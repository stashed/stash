/*
Copyright The Kmodules Authors.

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

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"gomodules.xyz/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
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

func GetVersionInfo(client discovery.DiscoveryInterface) (int64, int64, int64, string, string, error) {
	info, err := client.ServerVersion()
	if err != nil {
		return -1, -1, -1, "", "", err
	}
	gv, err := version.NewVersion(info.GitVersion)
	if err != nil {
		return -1, -1, -1, "", "", err
	}
	v := gv.ToMutator().ResetMetadata().ResetPrerelease()
	return v.Major(), v.Minor(), v.Patch(), v.Prerelease(), v.Metadata(), nil
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
	return ExistsGroupVersionKind(client, groupVersion, kind)
}

func ExistsGroupVersionKind(client discovery.DiscoveryInterface, groupVersion, kind string) bool {
	if resourceList, err := client.ServerPreferredResources(); discovery.IsGroupDiscoveryFailedError(err) || err == nil {
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
				if resource.Kind == kind {
					return true
				}
			}
		}
	}
	return false
}

type KnownBug struct {
	URL string
	Fix string
}

func (e *KnownBug) Error() string {
	return "Bug: " + e.URL + ". To fix, " + e.Fix
}

var err62649_K1_9 = &KnownBug{URL: "https://github.com/kubernetes/kubernetes/pull/62649", Fix: "upgrade to Kubernetes 1.9.8 or later."}
var err62649_K1_10 = &KnownBug{URL: "https://github.com/kubernetes/kubernetes/pull/62649", Fix: "upgrade to Kubernetes 1.10.2 or later."}
var err83778_K1_16 = &KnownBug{URL: "https://github.com/kubernetes/kubernetes/pull/83787", Fix: "upgrade to Kubernetes 1.16.2 or later."}

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
	glog.Infof("Kubernetes version: %#v\n", info)

	gv, err := version.NewVersion(info.GitVersion)
	if err != nil {
		return err
	}
	v := gv.ToMutator().ResetMetadata().ResetPrerelease().Done()

	nodes, err := kc.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/master",
	})
	if err != nil {
		return err
	}
	multiMaster := len(nodes.Items) > 1

	return checkVersion(v, multiMaster, constraint, blackListedVersions, blackListedMultiMasterVersions)
}

func checkVersion(v *version.Version, multiMaster bool, constraint string, blackListedVersions map[string]error, blackListedMultiMasterVersions map[string]error) error {
	vs := v.String()

	if constraint != "" {
		c, err := version.NewConstraint(constraint)
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
