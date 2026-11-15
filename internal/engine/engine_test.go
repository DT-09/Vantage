package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DT-09/Vantage/internal/journal"
	"github.com/DT-09/Vantage/internal/plan"
)

func TestExecuteRespectsDependencies(t *testing.T) {
	j, _, err := journal.Open(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil { t.Fatal(err) }
	defer j.Close()
	workflow := plan.Workflow{Tasks:[]plan.Task{
		{ID:"first",Kind:"echo"},
		{ID:"second",Kind:"echo",Needs:[]string{"first"}},
	}}
	h := Registry{"echo":func(_ context.Context, task plan.Task, deps map[string]Result) (Result,error) {
		if task.ID == "second" && deps["first"].Value != "first" { t.Fatalf("dependency result unavailable: %#v", deps) }
		return Result{Value:task.ID},nil
	}}
	report, err := Execute(context.Background(), workflow, Config{Workers:2,RunID:"test-run",Journal:j}, h)
	if err != nil { t.Fatal(err) }
	if report.Results["second"].Value != "second" { t.Fatalf("unexpected result: %#v", report.Results) }
}

func TestExecuteRejectsMissingHandler(t *testing.T) {
	j, _, err := journal.Open(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil { t.Fatal(err) }
	defer j.Close()
	_, err = Execute(context.Background(), plan.Workflow{Tasks:[]plan.Task{{ID:"x",Kind:"missing"}}}, Config{Workers:1,RunID:"r",Journal:j}, Registry{})
	if err == nil { t.Fatal("expected missing handler error") }
}
