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
	"sort"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// https://efficientcodeblog.wordpress.com/2018/02/15/finding-all-paths-between-two-nodes-in-a-graph/
// https://www.baeldung.com/cs/simple-paths-between-two-vertices
// FindPaths returns all simple paths betweek 2 vertices of a graph.
func FindPaths(graph *Graph, src, dst schema.GroupVersionResource) []*Path {
	var paths []*Path

	visited := map[schema.GroupVersionResource]bool{}
	curPath := &Path{
		Source:   src,
		Target:   dst,
		Distance: 0,
		Edges:    nil,
	}
	paths = dfs(graph, src, dst, visited, curPath, paths)

	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Distance < paths[j].Distance
	})
	return paths
}

func dfs(g *Graph, u, d schema.GroupVersionResource, visited map[schema.GroupVersionResource]bool, curPath *Path, paths []*Path) []*Path {
	visited[u] = true
	if u == d {
		copyPath := *curPath
		copyPath.Edges = make([]*Edge, len(curPath.Edges))
		for i := range curPath.Edges {
			copyPath.Edges[i] = curPath.Edges[i]
		}
		paths = append(paths, &copyPath)
		visited[u] = false
		return paths
	}

	for next, edge := range g.edges[u] {
		if !visited[next] {
			curPath.Edges = append(curPath.Edges, edge)
			curPath.Distance += edge.W
			paths = dfs(g, next, d, visited, curPath, paths)
			curPath.Edges = curPath.Edges[:len(curPath.Edges)-1]
			curPath.Distance -= edge.W
		}
	}
	visited[u] = false

	return paths
}
