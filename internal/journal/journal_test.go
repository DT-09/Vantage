package journal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendAndRecover(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	j, _, err := Open(path); if err != nil { t.Fatal(err) }
	if err := j.Append(Event{RunID:"r1",State:"run.started"}); err != nil { t.Fatal(err) }
	if err := j.Close(); err != nil { t.Fatal(err) }
	j, events, err := Open(path); if err != nil { t.Fatal(err) }
	defer j.Close()
	if len(events) != 1 || events[0].Sequence != 1 { t.Fatalf("unexpected recovery: %#v", events) }
}
func TestRejectCorruption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(path, []byte("{broken}\n"), 0600); err != nil { t.Fatal(err) }
	if _, _, err := Open(path); err == nil { t.Fatal("expected corruption error") }
}
