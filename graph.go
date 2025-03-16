package bubble

import "slices"

type graph[T comparable] struct {
	vertices []T
	edges    map[T][]T
	vis      map[T]struct{}
	stack    []T
}

func newGraph[T comparable]() *graph[T] {
	return &graph[T]{
		edges: make(map[T][]T),
		vis:   make(map[T]struct{}),
	}
}

func (g *graph[T]) addVertex(u T) {
	g.vertices = append(g.vertices, u)
}

func (g *graph[T]) addEdge(u T, v T) {
	g.edges[u] = append(g.edges[u], v)
}

func (g *graph[T]) traverse(u T) {
	g.vis[u] = struct{}{}
	for _, v := range g.edges[u] {
		if _, ok := g.vis[v]; !ok {
			g.traverse(v)
		}
	}
	g.stack = append(g.stack, u)
}

func (g *graph[T]) sort(cmp func(T, T) int) []T {
	slices.SortStableFunc(g.vertices, cmp)
	for _, v := range g.vertices {
		if _, ok := g.vis[v]; !ok {
			g.traverse(v)
		}
	}
	return g.stack
}
