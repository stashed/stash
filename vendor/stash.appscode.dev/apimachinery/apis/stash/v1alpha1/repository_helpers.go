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

package v1alpha1

import (
	"stash.appscode.dev/apimachinery/crds"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/apiextensions"
)

func (_ Repository) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crds.MustCustomResourceDefinition(SchemeGroupVersion.WithResource(ResourcePluralRepository))
}

func (r *Repository) LocalNetworkVolume() bool {
	if r.Spec.Backend.Local != nil &&
		r.Spec.Backend.Local.NFS != nil {
		return true
	}
	return false
}

func (r *Repository) LocalNetworkVolumePath() string {
	if r.Spec.Backend.Local != nil {
		if r.Spec.Backend.Local.NFS != nil {
			return r.Spec.Backend.Local.NFS.Path
		}
	}
	return ""
}

func (r *Repository) UsageAllowed(srcNamespace *core.Namespace) bool {
	if r.Spec.UsagePolicy == nil {
		return r.Namespace == srcNamespace.Name
	}
	return r.isNamespaceAllowed(srcNamespace)
}

func (r *Repository) isNamespaceAllowed(srcNamespace *core.Namespace) bool {
	allowedNamespaces := r.Spec.UsagePolicy.AllowedNamespaces

	if allowedNamespaces.From == nil {
		return false
	}

	if *allowedNamespaces.From == NamespacesFromAll {
		return true
	}

	if *allowedNamespaces.From == NamespacesFromSame {
		return r.Namespace == srcNamespace.Name
	}

	return selectorMatches(allowedNamespaces.Selector, srcNamespace.Labels)
}

func selectorMatches(ls *metav1.LabelSelector, srcLabels map[string]string) bool {
	selector, err := metav1.LabelSelectorAsSelector(ls)
	if err != nil {
		klog.Infoln("invalid label selector: ", ls)
		return false
	}
	return selector.Matches(labels.Set(srcLabels))
}
