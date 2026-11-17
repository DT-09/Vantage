package journal

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Event struct {
	Sequence uint64    `json:"sequence"`
	RunID    string    `json:"run_id"`
	TaskID   string    `json:"task_id,omitempty"`
	State    string    `json:"state"`
	At       time.Time `json:"at"`
	Detail   string    `json:"detail,omitempty"`
}

type record struct {
	Event    Event  `json:"event"`
	Checksum string `json:"checksum"`
}

type Journal struct {
	mu       sync.Mutex
	file     *os.File
	sequence uint64
	failed   error
}

func Open(path string) (*Journal, []Event, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, nil, err
	}
	events, err := readAll(f)
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	var seq uint64
	if len(events) > 0 {
		seq = events[len(events)-1].Sequence
	}
	return &Journal{file: f, sequence: seq}, events, nil
}

func checksum(e Event) (string, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func readAll(f *os.File) ([]Event, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var out []Event
	var expected uint64 = 1
	for s.Scan() {
		var r record
		if err := json.Unmarshal(s.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("journal corrupt at record %d: %w", expected, err)
		}
		sum, err := checksum(r.Event)
		if err != nil {
			return nil, err
		}
		if sum != r.Checksum {
			return nil, fmt.Errorf("journal checksum mismatch at sequence %d", expected)
		}
		if r.Event.Sequence != expected {
			return nil, fmt.Errorf("journal sequence gap: expected %d, got %d", expected, r.Event.Sequence)
		}
		out = append(out, r.Event)
		expected++
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (j *Journal) Append(e Event) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.failed != nil {
		return fmt.Errorf("journal is unavailable after a prior write failure: %w", j.failed)
	}
	e.Sequence = j.sequence + 1
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	sum, err := checksum(e)
	if err != nil {
		return err
	}
	b, err := json.Marshal(record{Event: e, Checksum: sum})
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if _, err = j.file.Write(b); err != nil {
		j.failed = err
		return err
	}
	if err = j.file.Sync(); err != nil {
		j.failed = err
		return err
	}
	j.sequence = e.Sequence
	return nil
}

func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.file.Close()
}
