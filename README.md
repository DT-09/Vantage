# VANTAGE

**Invoice-to-payment reconciliation for B2B SaaS finance teams.**

Vantage reviews customer-provided invoice and payment CSV exports, matches successful payments to invoices, and produces a traceable exception worklist. It is a read-only review: it does not connect to production accounts, change financial records, initiate collections, or claim that a discrepancy is a confirmed loss.

## The paid problem

Finance teams need to know whether invoice amounts recorded as paid agree with successful payment records. A mismatch can mean an export issue, timing difference, partial payment, duplicate record, currency problem, or a genuine exception. Manual spreadsheet checks are slow and easy to repeat inconsistently.

Vantage makes the comparison reproducible and gives the reviewer source identifiers and arithmetic to investigate.

## Current MVP

- Loads invoice and payment CSV exports using a documented schema.
- Counts only successful payment statuses: `succeeded`, `paid`, and `captured`.
- Flags orphan payments, duplicate identifiers, currency mismatches, paid-amount mismatches, paid invoices below their amount due, and paid amounts exceeding invoice due.
- Uses integer minor units for arithmetic; no floating-point currency calculations.
- Writes `exceptions.csv` and `summary.json`.
- Never labels an exception as recovered revenue or a confirmed loss.

## Run a review

Requirements: Go 1.23+.

```sh
go test ./...
go vet ./...
go run ./cmd/vantage-reconcile \
  -invoices examples/revenue-review/invoices.csv \
  -payments examples/revenue-review/payments.csv \
  -out ./vantage-review
```

Output:
- `vantage-review/exceptions.csv`
- `vantage-review/summary.json`

## CSV contract

See [docs/csv-contract.md](docs/csv-contract.md) for required columns, status handling, amount format, and sample files.

## Commercial offer

Vantage's initial service is a **read-only invoice-to-payment exception review** for B2B SaaS companies using exported billing and payment data.

- Paid pilot: **$300 fixed fee**, capped at two hours of review and reporting.
- Follow-on review: **$500 per monthly period**, scoped before work begins.
- The effective hourly threshold is met only if actual delivery time stays within the agreed cap. No revenue or savings outcome is promised.

The pilot is not an accounting opinion, audit, tax service, ASC 606 / IFRS 15 assessment, payment recovery service, or substitute for the customer's finance team. See [commercial/offer.md](commercial/offer.md).

## Current limits — read before use

This is an early MVP, not production-grade financial software. It supports a narrow CSV contract and two-decimal currencies. It does not yet support refunds, disputes, chargebacks, processor fees, FX conversion, partial-payment allocation rules, credit notes, subscription entitlements, accounting ledgers, or automated source-system integrations. It does not establish why a mismatch exists. A qualified customer-side finance owner must review every exception before acting.

Do not upload card data, bank credentials, passwords, API keys, personal identifiers not needed for matching, or confidential data without an approved data-processing and security arrangement. Use synthetic or appropriately redacted exports for evaluation.

## Existing workflow runtime

The original deterministic Go workflow engine remains in `cmd/vantage` and `internal/engine`. It is separate from the reconciliation MVP.

## Build and test

```sh
go test ./...
go vet ./...
go build -trimpath -o bin/vantage ./cmd/vantage
go build -trimpath -o bin/vantage-reconcile ./cmd/vantage-reconcile
```

## License

MIT.
