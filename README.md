# VANTAGE

A deterministic, fault-aware workflow execution engine written in Go.

VANTAGE plans dependency graphs, runs registered tasks concurrently, persists lifecycle events to a durable journal, and verifies completed outputs. It is designed as an embeddable Go library with a small operator CLI.

## Status

Initial repository bootstrap. The implementation is being built around a standard-library-only core.

## Design principles

- Explicit task identity and dependency graphs.
- Deterministic planning and stable execution ordering.
- Bounded concurrency and cooperative cancellation.
- Append-only, checksummed execution journal.
- Fail-closed recovery when journal integrity is uncertain.
- No shell evaluation and no implicit execution of untrusted input.

## Requirements

- Go 1.23 or newer

## License

MIT
