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

package hub

import (
	"context"
	"fmt"
	"io/fs"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"kmodules.xyz/apiversion"
	kmapi "kmodules.xyz/client-go/api/v1"
	disco_util "kmodules.xyz/client-go/discovery"
	"kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub/resourceclasses"
	"kmodules.xyz/resource-metadata/hub/resourcedescriptors"

	stringz "gomodules.xyz/x/strings"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type HelmVersion string

const (
	HelmUnused HelmVersion = ""
	Helm2      HelmVersion = "Helm 2"
	Helm3      HelmVersion = "Helm 3"
	crdIconSVG             = "https://cdn.appscode.com/k8s/icons/apiextensions.k8s.io/customresourcedefinitions.svg"
)

// ttl for cached preferred list
const ttl = 5 * time.Minute

type Registry struct {
	uid   string
	helm  HelmVersion
	cache KV
	m     sync.RWMutex
	// TODO: store in KV so cached for multiple instances of BB api server
	cfg           *rest.Config
	preferred     map[schema.GroupResource]schema.GroupVersionResource
	lastRefreshed time.Time
	regGVK        map[schema.GroupVersionKind]*kmapi.ResourceID
	regGVR        map[schema.GroupVersionResource]*kmapi.ResourceID
}

var _ disco_util.ResourceMapper = &Registry{}

func NewRegistry(uid string, helm HelmVersion, cache KV) *Registry {
	r := &Registry{
		uid:    uid,
		helm:   helm,
		cache:  cache,
		regGVK: map[schema.GroupVersionKind]*kmapi.ResourceID{},
		regGVR: map[schema.GroupVersionResource]*kmapi.ResourceID{},
	}

	guess := make(map[schema.GroupResource]string)

	r.cache.Visit(func(key string, val *v1alpha1.ResourceDescriptor) {
		v := val.Spec.Resource // copy
		r.regGVK[v.GroupVersionKind()] = &v
		r.regGVR[v.GroupVersionResource()] = &v

		gr := v.GroupResource()
		if curVer, ok := guess[gr]; !ok || apiversion.MustCompare(v.Version, curVer) > 0 {
			guess[gr] = v.Version
		}
	})

	r.preferred = make(map[schema.GroupResource]schema.GroupVersionResource)
	for gr, ver := range guess {
		r.preferred[gr] = gr.WithVersion(ver)
	}

	return r
}

func NewRegistryOfKnownResources() *Registry {
	return NewRegistry(KnownUID, Helm3, KnownResources)
}

func (r *Registry) DiscoverResources(cfg *rest.Config) error {
	r.m.Lock()
	defer r.m.Unlock()

	r.cfg = cfg
	return r.discoverResources()
}

func (r *Registry) discoverResources() error {
	preferred, reg, err := r.createRegistry(r.cfg)
	if err != nil {
		return err
	}

	r.preferred = preferred
	r.lastRefreshed = time.Now()
	for filename, rd := range reg {
		if _, found := r.cache.Get(filename); !found {
			r.regGVK[rd.Spec.Resource.GroupVersionKind()] = &rd.Spec.Resource
			r.regGVR[rd.Spec.Resource.GroupVersionResource()] = &rd.Spec.Resource
			r.cache.Set(filename, rd)
		}
	}

	return nil
}

func (r *Registry) Refresh(cfg *rest.Config) error {
	if time.Since(r.lastRefreshed) > ttl {
		return r.DiscoverResources(cfg)
	}
	return nil
}

func (r *Registry) Reset() {
	r.m.Lock()
	defer r.m.Unlock()
	if r.cfg == nil {
		return
	}
	if err := r.discoverResources(); err != nil {
		klog.ErrorS(err, "failed to reset Registry")
	}
}

func DiscoverHelm(cfg *rest.Config) (HelmVersion, string, error) {
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return HelmUnused, "", err
	}

	services, err := kc.CoreV1().Services(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", "tiller-deploy").String(),
	})
	if err != nil {
		return HelmUnused, "", err
	}

	if len(services.Items) == 1 {
		return Helm2, services.Items[0].Namespace, nil
	}
	return Helm3, "", nil
}

func (r *Registry) SetHelmVersion(helm HelmVersion) {
	r.helm = helm
}

func (r *Registry) Register(gvr schema.GroupVersionResource, cfg *rest.Config) error {
	r.m.RLock()
	if _, found := r.regGVR[gvr]; found {
		r.m.RUnlock()
		return nil
	}
	r.m.RUnlock()

	return r.DiscoverResources(cfg)
}

func (r *Registry) createRegistry(cfg *rest.Config) (map[schema.GroupResource]schema.GroupVersionResource, map[string]*v1alpha1.ResourceDescriptor, error) {
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	apiext, err := crd_cs.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	rsLists, err := kc.Discovery().ServerPreferredResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, nil, err
	}

	reg := make(map[string]*v1alpha1.ResourceDescriptor)
	for _, rsList := range rsLists {
		for i := range rsList.APIResources {
			rs := rsList.APIResources[i]

			// skip sub resource
			if strings.ContainsRune(rs.Name, '/') {
				continue
			}
			// if resource can't be listed or read (get) skip it
			if !stringz.Contains(rs.Verbs, "list") || !stringz.Contains(rs.Verbs, "get") {
				continue
			}

			gv, err := schema.ParseGroupVersion(rsList.GroupVersion)
			if err != nil {
				return nil, nil, err
			}
			rs.Group = gv.Group
			rs.Version = gv.Version

			scope := kmapi.ClusterScoped
			if rs.Namespaced {
				scope = kmapi.NamespaceScoped
			}

			filename := fmt.Sprintf("%s/%s/%s.yaml", rs.Group, rs.Version, rs.Name)
			rd := v1alpha1.ResourceDescriptor{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Kind:       v1alpha1.ResourceKindResourceDescriptor,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s-%s-%s", rs.Group, rs.Version, rs.Name),
					Labels: map[string]string{
						"k8s.io/group":    rs.Group,
						"k8s.io/version":  rs.Version,
						"k8s.io/resource": rs.Name,
						"k8s.io/kind":     rs.Kind,
					},
				},
				Spec: v1alpha1.ResourceDescriptorSpec{
					Resource: kmapi.ResourceID{
						Group:   rs.Group,
						Version: rs.Version,
						Name:    rs.Name,
						Kind:    rs.Kind,
						Scope:   scope,
					},
				},
			}
			if !v1alpha1.IsOfficialType(rd.Spec.Resource.Group) {
				crd, err := apiext.CustomResourceDefinitions().Get(context.TODO(), fmt.Sprintf("%s.%s", rd.Spec.Resource.Name, rd.Spec.Resource.Group), metav1.GetOptions{})
				if err == nil {
					var inner apiextensions.CustomResourceDefinition
					err = crdv1beta1.Convert_v1beta1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(crd, &inner, nil)
					if err != nil {
						return nil, nil, err
					}

					var out crdv1.CustomResourceDefinition
					err = crdv1.Convert_apiextensions_CustomResourceDefinition_To_v1_CustomResourceDefinition(&inner, &out, nil)
					if err != nil {
						return nil, nil, err
					}

					for _, v := range out.Spec.Versions {
						if v.Name == rs.Version {
							rd.Spec.Validation = v.Schema
							break
						}
					}
				}
			}
			reg[filename] = &rd
		}
	}

	preferred := make(map[schema.GroupResource]schema.GroupVersionResource)
	for _, rd := range reg {
		gvr := rd.Spec.Resource.GroupVersionResource()
		preferred[gvr.GroupResource()] = gvr
	}

	err = fs.WalkDir(resourcedescriptors.FS(), ".", func(filename string, e fs.DirEntry, err error) error {
		if !e.IsDir() {
			delete(reg, filename)
		}
		return err
	})
	return preferred, reg, err
}

func (r *Registry) Visit(f func(key string, val *v1alpha1.ResourceDescriptor)) {
	for _, gvr := range r.Resources() {
		key := r.filename(gvr)
		if rd, ok := r.cache.Get(key); ok {
			f(key, rd)
		}
	}
}

func (r *Registry) Missing(in schema.GroupVersionResource) bool {
	r.m.RLock()
	defer r.m.RUnlock()
	for _, gvr := range r.preferred {
		if gvr == in {
			return false
		}
	}
	return true
}

func (r *Registry) findGVR(in *v1alpha1.GroupResources, keepOfficialTypes bool) (schema.GroupVersionResource, bool) {
	r.m.RLock()
	defer r.m.RUnlock()
	for _, group := range in.Groups {
		if gvr, ok := r.preferred[schema.GroupResource{Group: group, Resource: in.Resource}]; ok {
			return gvr, true
		}
	}
	for _, group := range in.Groups {
		if keepOfficialTypes || !v1alpha1.IsOfficialType(group) {
			gvr, ok := LatestGVRs[schema.GroupResource{Group: group, Resource: in.Resource}]
			return gvr, ok
		}
	}
	return schema.GroupVersionResource{}, false
}

func (r *Registry) ResourceIDForGVK(gvk schema.GroupVersionKind) (*kmapi.ResourceID, error) {
	r.m.RLocker()
	defer r.m.RUnlock()
	return r.regGVK[gvk], nil
}

func (r *Registry) ResourceIDForGVR(gvr schema.GroupVersionResource) (*kmapi.ResourceID, error) {
	r.m.RLocker()
	defer r.m.RUnlock()
	return r.regGVR[gvr], nil
}

func (r *Registry) GVR(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	rid, exists := r.regGVK[gvk]
	if !exists {
		return schema.GroupVersionResource{}, UnregisteredErr{gvk.String()}
	}
	return rid.GroupVersionResource(), nil
}

func (r *Registry) TypeMeta(gvr schema.GroupVersionResource) (metav1.TypeMeta, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	rid, exists := r.regGVR[gvr]
	if !exists {
		return metav1.TypeMeta{}, UnregisteredErr{gvr.String()}
	}
	return rid.TypeMeta(), nil
}

func (r *Registry) GVK(gvr schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	rid, exists := r.regGVR[gvr]
	if !exists {
		return schema.GroupVersionKind{}, UnregisteredErr{gvr.String()}
	}
	return rid.GroupVersionKind(), nil
}

func (r *Registry) IsNamespaced(gvr schema.GroupVersionResource) (bool, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	rid, exist := r.regGVR[gvr]
	if !exist {
		return false, UnregisteredErr{gvr.String()}
	}
	return rid.Scope == kmapi.NamespaceScoped, nil
}

func (r *Registry) IsPreferred(gvr schema.GroupVersionResource) (bool, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	if preferred, exists := r.preferred[gvr.GroupResource()]; exists {
		return preferred == gvr, nil
	}
	return false, nil
}

func (r *Registry) Preferred(gvr schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	if preferred, exists := r.preferred[gvr.GroupResource()]; exists {
		return preferred, nil
	}
	return gvr, nil
}

func (r *Registry) ExistsGVR(gvr schema.GroupVersionResource) (bool, error) {
	if gvr.Version == "" {
		return false, fmt.Errorf("unspecificed version for %v", gvr.GroupResource())
	}
	r.m.RLock()
	defer r.m.RUnlock()
	_, exists := r.regGVR[gvr]
	return exists, nil
}

func (r *Registry) ExistsGVK(gvk schema.GroupVersionKind) (bool, error) {
	if gvk.Version == "" {
		return false, fmt.Errorf("unspecificed version for %v", gvk.GroupKind())
	}
	r.m.RLock()
	defer r.m.RUnlock()
	_, exists := r.regGVK[gvk]
	return exists, nil
}

func (r *Registry) Resources() []schema.GroupVersionResource {
	r.m.RLock()
	defer r.m.RUnlock()

	out := make([]schema.GroupVersionResource, len(r.preferred))
	for _, gvr := range r.preferred {
		out = append(out, gvr)
	}
	return out
}

func (r *Registry) LoadByGVR(gvr schema.GroupVersionResource) (*v1alpha1.ResourceDescriptor, error) {
	return r.LoadByFile(r.filename(gvr))
}

func (r *Registry) filename(gvr schema.GroupVersionResource) string {
	if gvr.Group == "" && gvr.Version == "v1" {
		return fmt.Sprintf("core/v1/%s.yaml", gvr.Resource)
	}
	return fmt.Sprintf("%s/%s/%s.yaml", gvr.Group, gvr.Version, gvr.Resource)
}

func (r *Registry) LoadByName(name string) (*v1alpha1.ResourceDescriptor, error) {
	filename := strings.Replace(name, "-", "/", 2) + ".yaml"
	return r.LoadByFile(filename)
}

func (r *Registry) LoadByFile(filename string) (*v1alpha1.ResourceDescriptor, error) {
	obj, ok := r.cache.Get(filename)
	if !ok {
		return nil, UnregisteredErr{filename}
	}
	return obj, nil
}

func (r *Registry) CompleteResourcePanel() (*v1alpha1.ResourcePanel, error) {
	return r.createResourcePanel(true)
}

func (r *Registry) DefaultResourcePanel() (*v1alpha1.ResourcePanel, error) {
	return r.createResourcePanel(false)
}

func (r *Registry) createResourcePanel(keepOfficialTypes bool) (*v1alpha1.ResourcePanel, error) {
	sections := make(map[string]*v1alpha1.PanelSection)
	existingGRs := map[schema.GroupResource]bool{}

	// first add the known required sections
	for group, rc := range KnownClasses {
		if !rc.IsRequired() && string(r.helm) != rc.Name {
			continue
		}

		section := &v1alpha1.PanelSection{
			Name:              rc.Name,
			ResourceClassInfo: rc.Spec.ResourceClassInfo,
			Weight:            rc.Spec.Weight,
		}
		for _, entry := range rc.Spec.Entries {
			pe := v1alpha1.PanelEntry{
				Name:       entry.Name,
				Path:       entry.Path,
				Required:   entry.Required,
				Icons:      entry.Icons,
				Namespaced: rc.Name == string(Helm3),
			}
			if entry.Type != nil {
				gvr, ok := r.findGVR(entry.Type, keepOfficialTypes)
				if !ok {
					continue
				}
				pe.Resource = &kmapi.ResourceID{
					Group:   gvr.Group,
					Version: gvr.Version,
					Name:    gvr.Resource,
				}
				existingGRs[gvr.GroupResource()] = true
				if rd, err := r.LoadByGVR(gvr); err == nil {
					pe.Resource = &rd.Spec.Resource
					pe.Namespaced = rd.Spec.Resource.Scope == kmapi.NamespaceScoped
					pe.Icons = rd.Spec.Icons
					pe.Missing = r.Missing(gvr)
					pe.Installer = rd.Spec.Installer
				}
			}
			section.Entries = append(section.Entries, pe)
		}
		sections[group] = section
	}

	// now, auto discover sections from registry
	r.Visit(func(_ string, rd *v1alpha1.ResourceDescriptor) {
		if !keepOfficialTypes && v1alpha1.IsOfficialType(rd.Spec.Resource.Group) {
			return // skip k8s.io api groups
		}

		gvr := rd.Spec.Resource.GroupVersionResource()
		if _, found := existingGRs[gvr.GroupResource()]; found {
			return
		}

		section, found := sections[rd.Spec.Resource.Group]
		if !found {
			if rc, found := KnownClasses[rd.Spec.Resource.Group]; found {
				w := math.MaxInt16
				if rc.Spec.Weight > 0 {
					w = rc.Spec.Weight
				}
				section = &v1alpha1.PanelSection{
					Name:              rc.Name,
					ResourceClassInfo: rc.Spec.ResourceClassInfo,
					Weight:            w,
				}
			} else {
				// unknown api group, so use CRD icon
				name := resourceclasses.ResourceClassName(rd.Spec.Resource.Group)
				section = &v1alpha1.PanelSection{
					Name: name,
					ResourceClassInfo: v1alpha1.ResourceClassInfo{
						APIGroup: rd.Spec.Resource.Group,
					},
					Weight: math.MaxInt16,
				}
			}
			sections[rd.Spec.Resource.Group] = section
		}

		section.Entries = append(section.Entries, v1alpha1.PanelEntry{
			Name:       rd.Spec.Resource.Kind,
			Resource:   &rd.Spec.Resource,
			Icons:      rd.Spec.Icons,
			Namespaced: rd.Spec.Resource.Scope == kmapi.NamespaceScoped,
			Missing:    r.Missing(gvr),
			Installer:  rd.Spec.Installer,
		})
		existingGRs[gvr.GroupResource()] = true
	})

	return toPanel(sections)
}

func toPanel(in map[string]*v1alpha1.PanelSection) (*v1alpha1.ResourcePanel, error) {
	sections := make([]*v1alpha1.PanelSection, 0, len(in))

	for key, section := range in {
		if !strings.HasSuffix(key, ".local") {
			sort.Slice(section.Entries, func(i, j int) bool {
				return section.Entries[i].Name < section.Entries[j].Name
			})
		}

		// Set icon for sections missing icon
		if len(section.Icons) == 0 {
			// TODO: Use a different icon for section
			section.Icons = []v1alpha1.ImageSpec{
				{
					Source: crdIconSVG,
					Type:   "image/svg+xml",
				},
			}
		}
		// set icons for entries missing icon
		for i := range section.Entries {
			if len(section.Entries[i].Icons) == 0 {
				section.Entries[i].Icons = []v1alpha1.ImageSpec{
					{
						Source: crdIconSVG,
						Type:   "image/svg+xml",
					},
				}
			}
		}

		sections = append(sections, section)
	}

	sort.Slice(sections, func(i, j int) bool {
		if sections[i].Weight == sections[j].Weight {
			return sections[i].Name < sections[j].Name
		}
		return sections[i].Weight < sections[j].Weight
	})

	return &v1alpha1.ResourcePanel{Sections: sections}, nil
}

type UnregisteredErr struct {
	t string
}

var _ error = UnregisteredErr{}

func (e UnregisteredErr) Error() string {
	return e.t + " isn't registered"
}
