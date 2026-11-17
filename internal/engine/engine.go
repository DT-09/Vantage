package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DT-09/Vantage/internal/journal"
	"github.com/DT-09/Vantage/internal/plan"
)

type Result struct {
	Value string `json:"value,omitempty"`
}
type Handler func(context.Context, plan.Task, map[string]Result) (Result, error)
type Registry map[string]Handler

type Config struct {
	Workers int
	RunID   string
	Journal *journal.Journal
}
type Report struct {
	RunID      string            `json:"run_id"`
	StartedAt  time.Time         `json:"started_at"`
	FinishedAt time.Time         `json:"finished_at"`
	Results    map[string]Result `json:"results"`
	Errors     map[string]string `json:"errors,omitempty"`
}

func Execute(ctx context.Context, workflow plan.Workflow, cfg Config, handlers Registry) (Report, error) {
	if cfg.Workers < 1 {
		return Report{}, fmt.Errorf("workers must be at least 1")
	}
	if cfg.RunID == "" {
		return Report{}, fmt.Errorf("run ID is required")
	}
	if cfg.Journal == nil {
		return Report{}, fmt.Errorf("journal is required")
	}
	p, err := plan.Build(workflow)
	if err != nil {
		return Report{}, err
	}
	for _, t := range workflow.Tasks {
		if handlers[t.Kind] == nil {
			return Report{}, fmt.Errorf("no handler registered for kind %q", t.Kind)
		}
	}
	r := Report{RunID: cfg.RunID, StartedAt: time.Now().UTC(), Results: map[string]Result{}, Errors: map[string]string{}}
	if err := cfg.Journal.Append(journal.Event{RunID: cfg.RunID, State: "run.started", Detail: workflow.Name}); err != nil {
		return r, err
	}
	defer func() { r.FinishedAt = time.Now().UTC() }()
	for _, level := range p.Levels {
		if err := ctx.Err(); err != nil {
			r.Errors["$run"] = err.Error()
			break
		}
		type outcome struct {
			id     string
			result Result
			err    error
		}
		sem := make(chan struct{}, cfg.Workers)
		out := make(chan outcome, len(level))
		var wg sync.WaitGroup
		for _, task := range level {
			task := task
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				if err := cfg.Journal.Append(journal.Event{RunID: cfg.RunID, TaskID: task.ID, State: "task.started"}); err != nil {
					out <- outcome{id: task.ID, err: err}
					return
				}
				deps := make(map[string]Result, len(task.Needs))
				for _, id := range task.Needs {
					deps[id] = r.Results[id]
				}
				result, err := handlers[task.Kind](ctx, task, deps)
				state, detail := "task.succeeded", ""
				if err != nil {
					state, detail = "task.failed", err.Error()
				}
				if jerr := cfg.Journal.Append(journal.Event{RunID: cfg.RunID, TaskID: task.ID, State: state, Detail: detail}); jerr != nil && err == nil {
					err = jerr
				}
				out <- outcome{id: task.ID, result: result, err: err}
			}()
		}
		wg.Wait()
		close(out)
		failed := false
		for item := range out {
			if item.err != nil {
				r.Errors[item.id] = item.err.Error()
				failed = true
			} else {
				r.Results[item.id] = item.result
			}
		}
		if failed {
			break
		}
	}
	r.FinishedAt = time.Now().UTC()
	state := "run.succeeded"
	detail := ""
	if len(r.Errors) > 0 {
		state, detail = "run.failed", fmt.Sprintf("%d error(s)", len(r.Errors))
	}
	if err := cfg.Journal.Append(journal.Event{RunID: cfg.RunID, State: state, Detail: detail}); err != nil {
		return r, err
	}
	if len(r.Errors) > 0 {
		return r, fmt.Errorf("workflow failed: %d error(s)", len(r.Errors))
	}
	return r, nil
}
