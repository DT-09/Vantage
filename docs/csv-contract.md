# Vantage CSV contract (MVP)

Vantage reads two UTF-8 CSV files. Column names are exact and case-sensitive.

## invoices.csv

Required columns:

`invoice_id,customer_id,subscription_id,currency,amount_due,amount_paid,status`

## payments.csv

Required columns:

`payment_id,invoice_id,currency,amount,status`

## Rules

- Amounts are non-negative decimal strings with at most two decimal places (for example `125`, `125.5`, `125.50`). Values are converted to integer minor units.
- Use the same three-letter currency code on linked records. This MVP does not perform FX conversion.
- Only payment statuses `succeeded`, `paid`, and `captured` count toward the successful-payment total. Pending, failed, canceled, and unknown statuses do not count.
- Invoice status is compared case-insensitively. A status of `paid` with amount_paid below amount_due is flagged.
- Every successful payment is summed against its invoice. A difference from invoice amount_paid is flagged for review.
- Duplicate identifiers and payments that reference missing invoices are flagged.

## Outputs

`exceptions.csv` includes severity, code, invoice/payment identifiers, currency, amount_minor, description, and evidence. `summary.json` includes input counts and counts by exception code.

## Important limits

This is not a complete processor reconciliation or accounting system. The MVP does not model payment intent lifecycle transitions, refunds, disputes, chargebacks, fees, taxes, credit notes, partial allocation rules, FX, zero-/three-decimal currencies, or revenue recognition. The two-decimal amount rule must be respected. The report identifies mismatches in supplied exports; it cannot determine whether an amount is recoverable or whether an exception is a real loss.
