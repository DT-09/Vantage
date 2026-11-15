# VANTAGE

A deterministic, fault-aware workflow execution engine written in Go.

VANTAGE turns a declarative task graph into a bounded concurrent run, records lifecycle transitions in a durable append-only journal, and emits a machine-readable execution report.

## Implemented

- Stable dependency planning with cycle, duplicate, self-reference, and missing-dependency rejection.
- Bounded concurrent execution across independent DAG levels.
- Registered Go handlers; workflow files never execute shell commands.
- Cooperative context cancellation.
- Append-only JSONL journal with monotonic sequence numbers, SHA-256 record checksums, fsync, and fail-closed recovery.
- CLI handlers for writing a workspace file, hashing it, and verifying a dependency result.
- Unit tests for planner ordering, invalid graphs, journal recovery, corruption detection, and engine dependency handling.

## Requirements

Go 1.23+.

## Build and test

    go test ./...
    go vet ./...
    go build -trimpath -o bin/vantage ./cmd/vantage

## Run the example

    go run ./cmd/vantage run -file examples/basic.json

The example writes out/report.txt, hashes it, and verifies the expected digest. State is stored in .vantage/events.jsonl by default.

## Workflow format

A workflow has a name and a list of tasks. Each task has a unique ID, a registered kind, optional dependency IDs, and string input values.

Supported CLI task kinds:

| Kind | Input | Output |
|---|---|---|
| write | path, content | Relative output path |
| sha256 | path | SHA-256 hex digest |
| verify | from, expected | verified or an error |

Paths must be relative and cannot lexically escape the process working directory. Treat the working directory as a trusted workspace; do not point it at directories writable by untrusted actors.

## Architecture

- internal/plan: validates and deterministically layers the dependency DAG.
- internal/engine: schedules ready tasks and coordinates handlers.
- internal/journal: serializes and verifies durable event records.
- cmd/vantage: CLI and built-in handlers.

## Guarantees and limits

VANTAGE currently guarantees deterministic planning, bounded task concurrency, ordered journal sequence assignment, and integrity checking of complete journal records. It does not yet guarantee exactly-once task effects, automatic replay of unfinished tasks, distributed execution, multi-process journal locking, or transactional rollback. A handler may complete its side effect before a process crash; handlers that need safe retries must be idempotent.

## License

MIT.
