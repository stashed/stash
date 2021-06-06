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
	"fmt"
	"time"

	kmapi "kmodules.xyz/client-go/api/v1"
	dynamicfactory "kmodules.xyz/client-go/dynamic/factory"
	"kmodules.xyz/client-go/tools/clientcache"
	"kmodules.xyz/resource-metadata/hub"

	"github.com/gregjones/httpcache"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

/*
- Handle when multiple version of resources are available
- How to handle preferred version path missing
*/

func GetConnectedGraph(config *rest.Config, reg *hub.Registry, srcGVR schema.GroupVersionResource, ref types.NamespacedName) ([]*Edge, error) {
	cfg := clientcache.ConfigFor(config, 5*time.Minute, httpcache.NewMemoryCache())

	if err := reg.Register(srcGVR, cfg); err != nil {
		return nil, err
	}
	rd, err := reg.LoadByGVR(srcGVR)
	if err != nil {
		return nil, err
	}

	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	f := dynamicfactory.New(dc)

	var src *unstructured.Unstructured
	if rd.Spec.Resource.Scope == kmapi.NamespaceScoped {
		if ref.Namespace == "" {
			return nil, fmt.Errorf("missing namespace query parameter for %s with name %s", srcGVR, ref.Name)
		}
		src, err = f.ForResource(srcGVR).Namespace(ref.Namespace).Get(ref.Name)
		if err != nil {
			return nil, err
		}
	} else {
		src, err = f.ForResource(srcGVR).Get(ref.Name)
		if err != nil {
			return nil, err
		}
	}

	g, err := LoadGraph(reg)
	if err != nil {
		return nil, err
	}
	realGraph, err := g.generateRealGraph(f, src)
	if err != nil {
		return nil, err
	}

	dist, prev := Dijkstra(realGraph, srcGVR)

	out := make([]*Edge, 0, len(prev))
	for target, edge := range prev {
		if target != srcGVR && edge != nil {
			out = append(out, &Edge{
				Src:        edge.Src,
				Dst:        edge.Dst,
				W:          dist[target],
				Connection: edge.Connection,
				Forward:    edge.Forward,
			})
		}
	}
	return out, nil
}

// getRealGraph runs BFS on the original graph and returns a graph that has real connection
// with the source resource.
func (g *Graph) generateRealGraph(f dynamicfactory.Factory, src *unstructured.Unstructured) (*Graph, error) {
	srcGVK := schema.FromAPIVersionAndKind(src.GetAPIVersion(), src.GetKind())
	srcGVR, err := g.r.GVR(srcGVK)
	if err != nil {
		return nil, err
	}
	objMap := map[schema.GroupVersionResource][]*unstructured.Unstructured{
		srcGVR: {src},
	}

	visited := map[schema.GroupVersionResource]bool{}
	realGraph := NewGraph(g.r)
	// Queue for the BSF
	q := make([]schema.GroupVersionResource, 0)

	// Push the source node
	q = append(q, srcGVR)
	visited[srcGVR] = true
	finder := ObjectFinder{
		Factory: f,
		Mapper:  g.r,
	}
	for {
		// Pop the first item
		u := q[0]
		q = q[1:]
		for v, e := range g.edges[u] {
			if !visited[v] {
				// Find the connected objects. The object might be connected via multiple paths.
				// Hence, we are checking connection from all the child object of u.
				srcObjects := objMap[u]
				var dstObjects []*unstructured.Unstructured
				for _, srcObj := range srcObjects {
					objects, err := finder.ResourcesFor(srcObj, e)
					if err != nil && !kerr.IsNotFound(err) {
						return nil, err
					}
					dstObjects = appendObjects(dstObjects, objects...)
				}
				if len(dstObjects) > 0 {
					// Real edge exists, we need to traverse. So, add it to the queue.
					q = append(q, v)
					realGraph.AddEdge(e)
					objMap[v] = dstObjects
					visited[v] = true
				}
			}
		}
		if len(q) == 0 {
			break
		}
	}
	return realGraph, nil
}
