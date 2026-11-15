package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DT-09/Vantage/internal/engine"
	"github.com/DT-09/Vantage/internal/journal"
	"github.com/DT-09/Vantage/internal/plan"
)

func main() {
	if len(os.Args) < 2 { usage(); os.Exit(2) }
	switch os.Args[1] {
	case "run": run(os.Args[2:])
	case "version": fmt.Println("vantage dev")
	default: usage(); os.Exit(2)
	}
}

func usage() { fmt.Fprintln(os.Stderr, "usage: vantage <run|version>") }

func run(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	file := fs.String("file", "workflow.json", "workflow definition")
	state := fs.String("state", ".vantage/events.jsonl", "durable event journal")
	workers := fs.Int("workers", 4, "maximum concurrent tasks")
	_ = fs.Parse(args)
	raw, err := os.ReadFile(*file); check(err)
	var w plan.Workflow
	check(json.Unmarshal(raw, &w))
	j, _, err := journal.Open(*state); check(err)
	defer j.Close()
	handlers := engine.Registry{"write":writeHandler, "sha256":hashHandler, "verify":verifyHandler}
	report, err := engine.Execute(context.Background(), w, engine.Config{Workers:*workers, RunID:randomID(), Journal:j}, handlers)
	enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); _ = enc.Encode(report)
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}

func safePath(name string) (string, error) {
	if name == "" || filepath.IsAbs(name) { return "", fmt.Errorf("path must be a non-empty relative path") }
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) { return "", fmt.Errorf("path escapes workspace") }
	return clean, nil
}

func writeHandler(ctx context.Context, t plan.Task, _ map[string]engine.Result) (engine.Result, error) {
	if err := ctx.Err(); err != nil { return engine.Result{}, err }
	p, err := safePath(t.Input["path"]); if err != nil { return engine.Result{}, err }
	if err := os.MkdirAll(filepath.Dir(p), 0750); err != nil { return engine.Result{}, err }
	if err := os.WriteFile(p, []byte(t.Input["content"]), 0600); err != nil { return engine.Result{}, err }
	return engine.Result{Value:p}, nil
}
func hashHandler(ctx context.Context, t plan.Task, _ map[string]engine.Result) (engine.Result, error) {
	if err := ctx.Err(); err != nil { return engine.Result{}, err }
	p, err := safePath(t.Input["path"]); if err != nil { return engine.Result{}, err }
	b, err := os.ReadFile(p); if err != nil { return engine.Result{}, err }
	sum := sha256.Sum256(b); return engine.Result{Value:hex.EncodeToString(sum[:])}, nil
}
func verifyHandler(ctx context.Context, t plan.Task, deps map[string]engine.Result) (engine.Result, error) {
	if err := ctx.Err(); err != nil { return engine.Result{}, err }
	actual := deps[t.Input["from"]].Value
	if actual == "" || actual != t.Input["expected"] { return engine.Result{}, fmt.Errorf("verification failed: expected %q, got %q", t.Input["expected"], actual) }
	return engine.Result{Value:"verified"}, nil
}
func randomID() string { b := make([]byte, 16); if _, err := os.ReadFile("/dev/urandom"); err == nil { /* ID fallback below is intentionally deterministic per process invocation. */ }; return fmt.Sprintf("%x", sha256.Sum256(append(b, byte(os.Getpid()))))[:16] }
func check(err error) { if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) } }
