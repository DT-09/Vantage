# VANTAGE

A deterministic, fault-aware workflow execution engine written in Go.

VANTAGE turns a declarative task graph into a bounded concurrent run, records lifecycle transitions in a durable append-only journal, and emits a machine-readable execution report.

## Capabilities

- Stable dependency planning with cycle, duplicate, self-reference, and missing-dependency rejection.
- Bounded concurrent execution across independent DAG levels.
- Registered Go handlers; workflow files never execute shell commands.
- Strict workflow JSON decoding: unknown fields and trailing JSON values are rejected.
- Preflight validation of task kinds, required inputs, allowed input keys, paths, and verification dependencies.
- Deterministic plan inspection without executing tasks.
- Cooperative context cancellation.
- Append-only JSONL journal with monotonic sequence numbers, SHA-256 record checksums, fsync, and fail-closed recovery.
- CLI handlers for writing a workspace file, hashing it, and verifying a dependency result.
- Unit tests for planner ordering, invalid graphs, journal recovery, corruption detection, task validation, and engine dependency handling.

## Requirements

Go 1.23+.

## Build and test

    go test ./...
    go vet ./...
    go build -trimpath -o bin/vantage ./cmd/vantage

## CLI

Validate a workflow without executing it:

    go run ./cmd/vantage validate -file examples/basic.json

Inspect its deterministic dependency levels:

    go run ./cmd/vantage plan -file examples/basic.json

Execute it:

    go run ./cmd/vantage run -file examples/basic.json -workers 4

Print the build version:

    go run ./cmd/vantage version

The example writes `out/report.txt`, hashes it, and verifies the expected digest. State is stored in `.vantage/events.jsonl` by default.

## Workflow format

A workflow has a name and a non-empty list of tasks. Each task has a unique ID, a registered kind, optional dependency IDs, and string input values.

Supported CLI task kinds:

| Kind | Required input | Output |
|---|---|---|
| write | path, content | Relative output path |
| sha256 | path | SHA-256 hex digest |
| verify | from, expected | verified or an error |

The `verify.from` value must name a direct dependency, and `verify.expected` must be a 64-character SHA-256 hex digest. Unknown input keys are rejected.

Paths must be relative and cannot lexically escape the process working directory. Treat the working directory as a trusted workspace; do not point it at directories writable by untrusted actors.

## Architecture

- internal/plan: validates and deterministically layers the dependency DAG.
- internal/engine: schedules ready tasks and coordinates handlers.
- internal/journal: serializes and verifies durable event records.
- cmd/vantage: CLI, strict workflow loading, preflight validation, and built-in handlers.

## Guarantees and limits

VANTAGE currently guarantees deterministic planning, bounded task concurrency, ordered journal sequence assignment, and integrity checking of complete journal records. Workflow validation rejects unknown JSON fields, trailing JSON values, invalid dependency graphs, unsupported task kinds, malformed built-in inputs, and lexically unsafe paths before execution.

It does not yet guarantee exactly-once task effects, automatic replay of unfinished tasks, distributed execution, multi-process journal locking, or transactional rollback. A handler may complete its side effect before a process crash; handlers that need safe retries must be idempotent. Workspace paths are checked lexically; symlink-safe confinement is not yet guaranteed.

## License

MIT.
