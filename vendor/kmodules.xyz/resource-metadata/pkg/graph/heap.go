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

// This example demonstrates a distance queue built using the heap interface.
package graph

import (
	"container/heap"
	"math"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// An Item is something we manage in a priority queue.
type Item struct {
	vertex schema.GroupVersionResource // The value of the item; arbitrary.
	dist   uint64                      // The priority of the item in the queue.
	// The index is needed by Update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

// A Queue implements heap.Interface and holds Items.
type Queue []*Item

func (q Queue) Len() int { return len(q) }

func (q Queue) Less(i, j int) bool {
	return q[i].dist < q[j].dist
}

func (q Queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *Queue) Push(x interface{}) {
	n := len(*q)
	item := x.(*Item)
	item.index = n
	*q = append(*q, item)
}

func (q *Queue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*q = old[0 : n-1]
	return item
}

// Update modifies the priority and value of an Item in the queue.
func (q *Queue) Update(item *Item, dist uint64) {
	item.dist = dist
	heap.Fix(q, item.index)
}

// ref: https://en.wikipedia.org/wiki/Dijkstra%27s_algorithm#Pseudocode

func Dijkstra(graph *Graph, src schema.GroupVersionResource) (dist map[schema.GroupVersionResource]uint64, prev map[schema.GroupVersionResource]*Edge) {
	dist = make(map[schema.GroupVersionResource]uint64)
	prev = make(map[schema.GroupVersionResource]*Edge)

	resources := graph.r.Resources()

	q := make(Queue, len(resources))
	i := 0
	items := make(map[schema.GroupVersionResource]*Item)
	for _, vertex := range resources {
		var d uint64 = math.MaxUint32 // avoid overflow
		if vertex == src {
			d = 0 // dist[src] = 0
		}

		dist[vertex] = d
		prev[vertex] = nil
		item := &Item{
			vertex: vertex,
			dist:   d,
			index:  i,
		}
		items[vertex] = item
		q[i] = item
		i++
	}
	heap.Init(&q)

	for len(q) > 0 {
		u := heap.Pop(&q).(*Item)

		for v, e := range graph.edges[u.vertex] {
			alt := dist[u.vertex] + e.W
			if alt < dist[v] {
				dist[v] = alt
				prev[v] = e
				q.Update(items[v], alt)
			}
		}
	}

	return
}

type Path struct {
	Source   schema.GroupVersionResource
	Target   schema.GroupVersionResource
	Distance uint64
	Edges    []*Edge
}

// v1 -> v2 -> v3
//func (p Path) String() string {
//	return strings.Join([]string{gvr.Group, "/", gvr.Version, ", Resource=", gvr.Resource}, "")
//}

func GeneratePaths(src schema.GroupVersionResource, dist map[schema.GroupVersionResource]uint64, prev map[schema.GroupVersionResource]*Edge) map[schema.GroupVersionResource]*Path {
	paths := make(map[schema.GroupVersionResource]*Path)

	for target, d := range dist {
		if d < math.MaxUint32 {
			path := Path{
				Source:   src,
				Target:   target,
				Distance: d,
				Edges:    make([]*Edge, 0),
			}

			u := target
			for prev[u] != nil {
				path.Edges = append(path.Edges, nil)
				copy(path.Edges[1:], path.Edges[0:])
				path.Edges[0] = prev[u]
				u = prev[u].Src
			}

			paths[target] = &path
		}
	}

	return paths
}
