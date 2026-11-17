package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DT-09/Vantage/internal/engine"
	"github.com/DT-09/Vantage/internal/journal"
	"github.com/DT-09/Vantage/internal/plan"
)

var version = "dev"

type validationReport struct {
	Name       string     `json:"name"`
	TaskCount  int        `json:"task_count"`
	LevelCount int        `json:"level_count"`
	Levels     [][]string `json:"levels"`
}

func main() {
	if err := execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "vantage:", err)
		os.Exit(1)
	}
}

func execute(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: vantage <run|validate|plan|version>")
	}
	switch args[0] {
	case "version":
		fmt.Println(version)
		return nil
	case "validate":
		return validate(args[1:], false)
	case "plan":
		return validate(args[1:], true)
	case "run":
		return run(args[1:])
	default:
		return fmt.Errorf("unknown command %q (usage: vantage <run|validate|plan|version>)", args[0])
	}
}

func loadWorkflow(path string) (plan.Workflow, plan.Plan, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return plan.Workflow{}, plan.Plan{}, fmt.Errorf("read workflow: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var workflow plan.Workflow
	if err := dec.Decode(&workflow); err != nil {
		return plan.Workflow{}, plan.Plan{}, fmt.Errorf("decode workflow: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return plan.Workflow{}, plan.Plan{}, errors.New("workflow contains more than one JSON value")
		}
		return plan.Workflow{}, plan.Plan{}, fmt.Errorf("invalid trailing data: %w", err)
	}
	if strings.TrimSpace(workflow.Name) == "" {
		return plan.Workflow{}, plan.Plan{}, errors.New("workflow name cannot be empty")
	}
	if len(workflow.Tasks) == 0 {
		return plan.Workflow{}, plan.Plan{}, errors.New("workflow must contain at least one task")
	}
	p, err := plan.Build(workflow)
	if err != nil {
		return plan.Workflow{}, plan.Plan{}, fmt.Errorf("invalid workflow: %w", err)
	}
	for _, task := range workflow.Tasks {
		if err := validateTask(task); err != nil {
			return plan.Workflow{}, plan.Plan{}, err
		}
	}
	return workflow, p, nil
}

func validateTask(task plan.Task) error {
	allowed := map[string]bool{}
	switch task.Kind {
	case "write":
		allowed = map[string]bool{"path": true, "content": true}
		if _, ok := task.Input["content"]; !ok {
			return fmt.Errorf("task %q: write requires a content input", task.ID)
		}
	case "sha256":
		allowed = map[string]bool{"path": true}
	case "verify":
		allowed = map[string]bool{"from": true, "expected": true}
		from := task.Input["from"]
		if from == "" {
			return fmt.Errorf("task %q: verify requires a from input", task.ID)
		}
		found := false
		for _, dependency := range task.Needs {
			if dependency == from {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("task %q: verify input from %q must name a direct dependency", task.ID, from)
		}
		expected, err := hex.DecodeString(task.Input["expected"])
		if err != nil || len(expected) != sha256.Size {
			return fmt.Errorf("task %q: verify expected must be a 64-character SHA-256 hex digest", task.ID)
		}
	default:
		return fmt.Errorf("task %q uses unsupported kind %q", task.ID, task.Kind)
	}
	for key := range task.Input {
		if !allowed[key] {
			return fmt.Errorf("task %q: unsupported input %q for kind %q", task.ID, key, task.Kind)
		}
	}
	if task.Kind == "write" || task.Kind == "sha256" {
		if _, err := safePath(task.Input["path"]); err != nil {
			return fmt.Errorf("task %q: %w", task.ID, err)
		}
	}
	return nil
}

func validate(args []string, showPlan bool) error {
	name := "validate"
	if showPlan {
		name = "plan"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	file := fs.String("file", "workflow.json", "workflow definition")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	workflow, p, err := loadWorkflow(*file)
	if err != nil {
		return err
	}
	report := validationReport{Name: workflow.Name, TaskCount: len(workflow.Tasks), LevelCount: len(p.Levels)}
	for _, level := range p.Levels {
		ids := make([]string, 0, len(level))
		for _, task := range level {
			ids = append(ids, task.ID)
		}
		report.Levels = append(report.Levels, ids)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func run(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	file := fs.String("file", "workflow.json", "workflow definition")
	state := fs.String("state", ".vantage/events.jsonl", "durable event journal")
	workers := fs.Int("workers", 4, "maximum concurrent tasks")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if *workers < 1 {
		return errors.New("workers must be at least 1")
	}
	workflow, _, err := loadWorkflow(*file)
	if err != nil {
		return err
	}
	j, _, err := journal.Open(*state)
	if err != nil {
		return fmt.Errorf("open journal: %w", err)
	}
	defer j.Close()

	handlers := engine.Registry{"write": writeHandler, "sha256": hashHandler, "verify": verifyHandler}
	report, executeErr := engine.Execute(context.Background(), workflow, engine.Config{
		Workers: *workers, RunID: randomID(), Journal: j,
	}, handlers)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode execution report: %w", err)
	}
	if executeErr != nil {
		return executeErr
	}
	return nil
}

func safePath(name string) (string, error) {
	if name == "" || filepath.IsAbs(name) {
		return "", errors.New("path must be a non-empty relative path")
	}
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes workspace")
	}
	return clean, nil
}

func writeHandler(ctx context.Context, task plan.Task, _ map[string]engine.Result) (engine.Result, error) {
	if err := ctx.Err(); err != nil {
		return engine.Result{}, err
	}
	path, err := safePath(task.Input["path"])
	if err != nil {
		return engine.Result{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return engine.Result{}, err
	}
	if err := os.WriteFile(path, []byte(task.Input["content"]), 0600); err != nil {
		return engine.Result{}, err
	}
	return engine.Result{Value: path}, nil
}

func hashHandler(ctx context.Context, task plan.Task, _ map[string]engine.Result) (engine.Result, error) {
	if err := ctx.Err(); err != nil {
		return engine.Result{}, err
	}
	path, err := safePath(task.Input["path"])
	if err != nil {
		return engine.Result{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return engine.Result{}, err
	}
	sum := sha256.Sum256(data)
	return engine.Result{Value: hex.EncodeToString(sum[:])}, nil
}

func verifyHandler(ctx context.Context, task plan.Task, deps map[string]engine.Result) (engine.Result, error) {
	if err := ctx.Err(); err != nil {
		return engine.Result{}, err
	}
	actual := deps[task.Input["from"]].Value
	if actual == "" || actual != task.Input["expected"] {
		return engine.Result{}, fmt.Errorf("verification failed: expected %q, got %q", task.Input["expected"], actual)
	}
	return engine.Result{Value: "verified"}, nil
}

func randomID() string {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		panic(fmt.Errorf("generate run ID: %w", err))
	}
	return hex.EncodeToString(data)
}
