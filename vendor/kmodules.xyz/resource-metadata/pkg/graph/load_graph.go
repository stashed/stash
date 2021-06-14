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

package graph

import (
	"kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

func LoadGraphOfKnownResources() (*Graph, error) {
	return LoadGraph(hub.NewRegistryOfKnownResources())
}

func LoadGraph(r *hub.Registry) (*Graph, error) {
	graph := NewGraph(r)

	var errs []error
	graph.r.Visit(func(key string, r *v1alpha1.ResourceDescriptor) {
		if err := addRDConnectionsToGraph(graph, r); err != nil {
			errs = append(errs, err)
		}
	})
	if len(errs) > 0 {
		return nil, utilerrors.NewAggregate(errs)
	}

	return graph, nil
}

func addRDConnectionsToGraph(graph *Graph, rd *v1alpha1.ResourceDescriptor) error {
	src := rd.Spec.Resource.GroupVersionResource()
	for _, conn := range rd.Spec.Connections {
		dst := conn.Target
		dstGVR, err := graph.r.GVR(dst.GroupVersionKind())
		if err != nil {
			return err
		}

		//var w uint64 = 1
		//if conn.ResourceConnectionSpec.Type == v1alpha1.MatchSelector &&
		//	conn.TargetLabelPath != "" &&
		//	strings.Trim(conn.TargetLabelPath, ".") != "metadata.labels" {
		//	w = 1 + CostFactorOfInAppFiltering
		//}

		graph.AddEdge(&Edge{
			Src:        src,
			Dst:        dstGVR,
			W:          getWeight(conn.Type),
			Connection: conn.ResourceConnectionSpec,
			Forward:    true,
		})

		backEdge := &Edge{
			Src:        dstGVR,
			Dst:        src,
			W:          getWeight(conn.Type),
			Connection: conn.ResourceConnectionSpec,
			Forward:    false,
		}
		//switch conn.Type {
		//case v1alpha1.MatchName:
		//	backEdge.W = 1
		//case v1alpha1.MatchSelector, v1alpha1.OwnedBy:
		//	backEdge.W = 1 + CostFactorOfInAppFiltering
		//case v1alpha1.MatchRef:
		//	backEdge.W = 1 + CostFactorOfInAppFiltering<<1
		//}

		graph.AddEdge(backEdge)
	}
	return nil
}

func getWeight(connType v1alpha1.ConnectionType) uint64 {
	switch connType {
	case v1alpha1.MatchName:
		return 1
	case v1alpha1.OwnedBy:
		return 2
	case v1alpha1.MatchSelector:
		return 4
	case v1alpha1.MatchRef:
		return 4
	}
	return 99999999
}
