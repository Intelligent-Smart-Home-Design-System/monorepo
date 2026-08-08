package engine

import "sort"

// FindDependencyCycle returns one closed dependency chain or nil when the graph is acyclic.
func FindDependencyCycle(dependencies map[string][]string) []string {
	const (
		unvisited = iota
		visiting
		visited
	)

	state := make(map[string]int, len(dependencies))
	path := make([]string, 0, len(dependencies))
	pathIndex := make(map[string]int, len(dependencies))

	nodes := make([]string, 0, len(dependencies))
	for node := range dependencies {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)

	var visit func(string) []string
	visit = func(node string) []string {
		state[node] = visiting
		pathIndex[node] = len(path)
		path = append(path, node)

		targets := append([]string(nil), dependencies[node]...)
		sort.Strings(targets)
		for _, target := range targets {
			switch state[target] {
			case visiting:
				start := pathIndex[target]
				cycle := append([]string(nil), path[start:]...)
				return append(cycle, target)
			case unvisited:
				if cycle := visit(target); len(cycle) > 0 {
					return cycle
				}
			}
		}

		path = path[:len(path)-1]
		delete(pathIndex, node)
		state[node] = visited
		return nil
	}

	for _, node := range nodes {
		if state[node] == unvisited {
			if cycle := visit(node); len(cycle) > 0 {
				return cycle
			}
		}
	}

	return nil
}
