package config

import "sort"

// detectCycle returns the nodes participating in a dependency cycle, or nil if no cycle exists.
func detectCycle(steps []Step) []string {
	enabled := make(map[string]bool, len(steps))
	for _, step := range steps {
		if step.Enabled {
			enabled[step.ID] = true
		}
	}

	graph := make(map[string][]string, len(enabled))
	for _, step := range steps {
		if !enabled[step.ID] {
			continue
		}
		deps := make([]string, 0, len(step.DependsOn))
		for _, dep := range step.DependsOn {
			if enabled[dep] {
				deps = append(deps, dep)
			}
		}
		graph[step.ID] = deps
	}

	visiting := make(map[string]bool, len(steps))
	visited := make(map[string]bool, len(steps))
	var stack []string

	var cycle []string
	var dfs func(string) bool
	dfs = func(node string) bool {
		visiting[node] = true
		stack = append(stack, node)

		for _, dep := range graph[node] {
			if !visited[dep] {
				if visiting[dep] {
					idx := indexOf(stack, dep)
					if idx >= 0 {
						cycle = append([]string{}, stack[idx:]...)
						cycle = append(cycle, dep)
					}
					return true
				}
				if dfs(dep) {
					return true
				}
			}
		}

		visiting[node] = false
		visited[node] = true
		stack = stack[:len(stack)-1]
		return false
	}

	ids := make([]string, 0, len(enabled))
	for id := range enabled {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		if visited[id] {
			continue
		}
		if dfs(id) {
			break
		}
	}

	return cycle
}

func indexOf(slice []string, target string) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}
	return -1
}
