package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeWorkflow(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workflow.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadWorkflowAcceptsValidDefinition(t *testing.T) {
	path := writeWorkflow(t, `{"name":"sample","tasks":[{"id":"one","kind":"write","input":{"path":"out.txt","content":"ok"}}]}`)
	workflow, p, err := loadWorkflow(path)
	if err != nil {
		t.Fatal(err)
	}
	if workflow.Name != "sample" || len(p.Levels) != 1 || p.Levels[0][0].ID != "one" {
		t.Fatalf("unexpected workflow or plan: %#v %#v", workflow, p)
	}
}

func TestLoadWorkflowRejectsUnknownFields(t *testing.T) {
	path := writeWorkflow(t, `{"name":"sample","unexpected":true,"tasks":[]}`)
	if _, _, err := loadWorkflow(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field error, got %v", err)
	}
}

func TestLoadWorkflowRejectsTrailingJSON(t *testing.T) {
	path := writeWorkflow(t, `{"name":"sample","tasks":[]} {"name":"second","tasks":[]}`)
	if _, _, err := loadWorkflow(path); err == nil || !strings.Contains(err.Error(), "more than one JSON value") {
		t.Fatalf("expected trailing-value error, got %v", err)
	}
}

func TestLoadWorkflowRejectsUnsupportedTaskKind(t *testing.T) {
	path := writeWorkflow(t, `{"name":"sample","tasks":[{"id":"one","kind":"shell"}]}`)
	if _, _, err := loadWorkflow(path); err == nil || !strings.Contains(err.Error(), "unsupported kind") {
		t.Fatalf("expected unsupported-kind error, got %v", err)
	}
}

func TestLoadWorkflowRejectsEmptyName(t *testing.T) {
	path := writeWorkflow(t, `{"tasks":[]}`)
	if _, _, err := loadWorkflow(path); err == nil || !strings.Contains(err.Error(), "name cannot be empty") {
		t.Fatalf("expected empty-name error, got %v", err)
	}
}
