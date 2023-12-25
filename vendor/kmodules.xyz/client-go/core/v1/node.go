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

package v1

import (
	"context"

	meta_util "kmodules.xyz/client-go/meta"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/tools/pager"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchNode(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Node) *core.Node, opts metav1.PatchOptions) (*core.Node, kutil.VerbType, error) {
	cur, err := c.CoreV1().Nodes().Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating Node %s", meta.Name)
		out, err := c.CoreV1().Nodes().Create(ctx, transform(&core.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: core.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}), metav1.CreateOptions{
			DryRun:       opts.DryRun,
			FieldManager: opts.FieldManager,
		})
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchNode(ctx, c, cur, transform, opts)
}

func PatchNode(ctx context.Context, c kubernetes.Interface, cur *core.Node, transform func(*core.Node) *core.Node, opts metav1.PatchOptions) (*core.Node, kutil.VerbType, error) {
	return PatchNodeObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchNodeObject(ctx context.Context, c kubernetes.Interface, cur, mod *core.Node, opts metav1.PatchOptions) (*core.Node, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.Node{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	klog.V(3).Infof("Patching Node %s with %s", cur.Name, string(patch))
	out, err := c.CoreV1().Nodes().Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateNode(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Node) *core.Node, opts metav1.UpdateOptions) (result *core.Node, err error) {
	attempt := 0
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Nodes().Get(ctx, meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().Nodes().Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update Node %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update Node %s after %d attempts due to %v", meta.Name, attempt, err)
	}
	return
}

// NodeReady returns whether a node is ready.
func NodeReady(node core.Node) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type != core.NodeReady {
			continue
		}
		return cond.Status == core.ConditionTrue
	}
	return false
}

// IsMaster returns whether a node is a master.
func IsMaster(node core.Node) bool {
	_, ok17 := node.Labels["node-role.kubernetes.io/master"]
	role16, ok16 := node.Labels["kubernetes.io/role"]
	return ok17 || (ok16 && role16 == "master")
}

type Topology struct {
	Regions       map[string][]string
	TotalNodes    int
	InstanceTypes map[string]int

	LabelZone         string
	LabelRegion       string
	LabelInstanceType string

	// https://github.com/kubernetes/kubernetes/blob/v1.17.2/staging/src/k8s.io/api/core/v1/well_known_labels.go

	//LabelHostname = "kubernetes.io/hostname"
	//
	//LabelZoneFailureDomain       = "failure-domain.beta.kubernetes.io/zone"
	//LabelZoneRegion              = "failure-domain.beta.kubernetes.io/region"
	//LabelZoneFailureDomainStable = "topology.kubernetes.io/zone"
	//LabelZoneRegionStable        = "topology.kubernetes.io/region"
	//
	//LabelInstanceType       = "beta.kubernetes.io/instance-type"
	//LabelInstanceTypeStable = "node.kubernetes.io/instance-type"
}

func (t Topology) ConvertAffinity(affinity *core.Affinity) {
	if affinity == nil {
		return
	}

	if affinity.PodAffinity != nil {
		t.convertPodAffinityTerm(affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		t.convertWeightedPodAffinityTerm(affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	}

	if affinity.PodAntiAffinity != nil {
		t.convertPodAffinityTerm(affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		t.convertWeightedPodAffinityTerm(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	}
}

func isZoneKey(key string) bool {
	return key == core.LabelZoneFailureDomain || key == "topology.kubernetes.io/zone"
}

func isRegionKey(key string) bool {
	return key == core.LabelZoneRegion || key == "topology.kubernetes.io/region"
}

func isInstanceTypeKey(key string) bool {
	return key == core.LabelInstanceType || key == "node.kubernetes.io/instance-type"
}

func (t Topology) convertPodAffinityTerm(terms []core.PodAffinityTerm) {
	for i := range terms {
		if isZoneKey(terms[i].TopologyKey) {
			terms[i].TopologyKey = t.LabelZone
		} else if isRegionKey(terms[i].TopologyKey) {
			terms[i].TopologyKey = t.LabelRegion
		} else if isInstanceTypeKey(terms[i].TopologyKey) {
			terms[i].TopologyKey = t.LabelInstanceType
		}
	}
}

func (t Topology) convertWeightedPodAffinityTerm(terms []core.WeightedPodAffinityTerm) {
	for i := range terms {
		if isZoneKey(terms[i].PodAffinityTerm.TopologyKey) {
			terms[i].PodAffinityTerm.TopologyKey = t.LabelZone
		} else if isRegionKey(terms[i].PodAffinityTerm.TopologyKey) {
			terms[i].PodAffinityTerm.TopologyKey = t.LabelRegion
		} else if isInstanceTypeKey(terms[i].PodAffinityTerm.TopologyKey) {
			terms[i].PodAffinityTerm.TopologyKey = t.LabelInstanceType
		}
	}
}

func DetectTopology(ctx context.Context, mc metadata.Interface) (*Topology, error) {
	var topology Topology
	topology.TotalNodes = 0

	mapRegion := make(map[string]sets.Set[string])
	instances := make(map[string]int)
	first := true

	nc := mc.Resource(schema.GroupVersionResource{
		Version:  "v1",
		Resource: "nodes",
	})
	lister := pager.New(pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
		return nc.List(ctx, opts)
	}))
	err := lister.EachListItem(context.Background(), metav1.ListOptions{}, func(obj runtime.Object) error {
		topology.TotalNodes++

		m, err := meta.Accessor(obj)
		if err != nil {
			return err
		}

		labels := m.GetLabels()

		if first {
			if _, ok := labels[core.LabelZoneRegionStable]; ok {
				topology.LabelRegion = core.LabelZoneRegionStable
			} else {
				topology.LabelRegion = core.LabelZoneRegion
			}

			if _, ok := labels[core.LabelZoneFailureDomainStable]; ok {
				topology.LabelZone = core.LabelZoneFailureDomainStable
			} else {
				topology.LabelZone = core.LabelZoneFailureDomain
			}

			if _, ok := labels[core.LabelInstanceTypeStable]; ok {
				topology.LabelInstanceType = core.LabelInstanceTypeStable
			} else {
				topology.LabelInstanceType = core.LabelInstanceType
			}

			first = false
		}

		os, _ := meta_util.GetStringValueForKeys(labels, core.LabelOSStable, "beta.kubernetes.io/os")
		if os != "linux" {
			return nil
		}
		arch, _ := meta_util.GetStringValueForKeys(labels, core.LabelArchStable, "beta.kubernetes.io/arch")
		if arch != "amd64" {
			return nil
		}

		region, _ := meta_util.GetStringValueForKeys(labels, topology.LabelRegion)
		zone, _ := meta_util.GetStringValueForKeys(labels, topology.LabelZone)
		if _, ok := mapRegion[region]; !ok {
			mapRegion[region] = sets.Set[string]{}
		}
		mapRegion[region].Insert(zone)

		instance, _ := meta_util.GetStringValueForKeys(labels, topology.LabelInstanceType)
		if n, ok := instances[instance]; ok {
			instances[instance] = n + 1
		} else {
			instances[instance] = 1
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	regions := make(map[string][]string)
	for k, v := range mapRegion {
		regions[k] = sets.List(v)
	}
	topology.Regions = regions
	topology.InstanceTypes = instances

	return &topology, nil
}
