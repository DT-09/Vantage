package plan

import (
	"fmt"
	"sort"
)

type Task struct {
	ID    string            `json:"id"`
	Kind  string            `json:"kind"`
	Needs []string          `json:"needs,omitempty"`
	Input map[string]string `json:"input,omitempty"`
}

type Workflow struct {
	Name  string `json:"name"`
	Tasks []Task `json:"tasks"`
}

type Plan struct {
	Levels [][]Task
	ByID   map[string]Task
}

func Build(w Workflow) (Plan, error) {
	p := Plan{ByID: make(map[string]Task, len(w.Tasks))}
	degree := make(map[string]int, len(w.Tasks))
	next := make(map[string][]string, len(w.Tasks))
	for _, t := range w.Tasks {
		if t.ID == "" {
			return Plan{}, fmt.Errorf("task ID cannot be empty")
		}
		if _, ok := p.ByID[t.ID]; ok {
			return Plan{}, fmt.Errorf("duplicate task ID %q", t.ID)
		}
		if t.Kind == "" {
			return Plan{}, fmt.Errorf("task %q has no kind", t.ID)
		}
		p.ByID[t.ID] = t
	}
	for _, t := range w.Tasks {
		seen := map[string]bool{}
		for _, dep := range t.Needs {
			if dep == t.ID {
				return Plan{}, fmt.Errorf("task %q depends on itself", t.ID)
			}
			if _, ok := p.ByID[dep]; !ok {
				return Plan{}, fmt.Errorf("task %q depends on unknown task %q", t.ID, dep)
			}
			if seen[dep] {
				return Plan{}, fmt.Errorf("task %q repeats dependency %q", t.ID, dep)
			}
			seen[dep] = true
			degree[t.ID]++
			next[dep] = append(next[dep], t.ID)
		}
	}
	ready := make([]string, 0)
	for id := range p.ByID {
		if degree[id] == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	visited := 0
	for len(ready) > 0 {
		level := make([]Task, 0, len(ready))
		for _, id := range ready {
			level = append(level, p.ByID[id])
			visited++
		}
		p.Levels = append(p.Levels, level)
		var following []string
		for _, id := range ready {
			for _, child := range next[id] {
				degree[child]--
				if degree[child] == 0 {
					following = append(following, child)
				}
			}
		}
		sort.Strings(following)
		ready = following
	}
	if visited != len(p.ByID) {
		return Plan{}, fmt.Errorf("dependency cycle detected")
	}
	return p, nil
}
