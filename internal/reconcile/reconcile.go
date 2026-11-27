package reconcile

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Invoice struct {
	ID, CustomerID, SubscriptionID, Currency, Status string
	AmountDue, AmountPaid int64
}

type Payment struct {
	ID, InvoiceID, Currency, Status string
	Amount int64
}

type Exception struct {
	Severity string `json:"severity"`
	Code string `json:"code"`
	InvoiceID string `json:"invoice_id,omitempty"`
	PaymentID string `json:"payment_id,omitempty"`
	Currency string `json:"currency,omitempty"`
	AmountMinor int64 `json:"amount_minor,omitempty"`
	Description string `json:"description"`
	Evidence string `json:"evidence"`
}

type Summary struct {
	InvoiceCount int `json:"invoice_count"`
	PaymentCount int `json:"payment_count"`
	ExceptionCount int `json:"exception_count"`
	ByCode map[string]int `json:"by_code"`
	Disclaimer string `json:"disclaimer"`
}

func parseMoney(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" { return 0, fmt.Errorf("amount is empty") }
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	parts := strings.Split(s, ".")
	if len(parts) > 2 { return 0, fmt.Errorf("invalid decimal amount %q", s) }
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 { return 0, fmt.Errorf("invalid decimal amount %q", s) }
	frac := "00"
	if len(parts) == 2 { frac = parts[1] }
	if len(frac) > 2 { return 0, fmt.Errorf("only two decimal places are supported: %q", s) }
	frac = frac + strings.Repeat("0", 2-len(frac))
	cents, err := strconv.ParseInt(frac, 10, 64)
	if err != nil { return 0, fmt.Errorf("invalid decimal amount %q", s) }
	value := whole*100 + cents
	if neg { value = -value }
	if value < 0 { return 0, fmt.Errorf("negative amounts are not supported: %q", s) }
	return value, nil
}

func readCSV(path string, required []string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil { return nil, fmt.Errorf("%s: read header: %w", path, err) }
	index := map[string]int{}
	for i, h := range header { index[strings.TrimSpace(h)] = i }
	for _, name := range required {
		if _, ok := index[name]; !ok { return nil, fmt.Errorf("%s: missing required column %q", path, name) }
	}
	var rows []map[string]string
	line := 1
	for {
		record, err := r.Read()
		if err == io.EOF { break }
		line++
		if err != nil { return nil, fmt.Errorf("%s:%d: %w", path, line, err) }
		row := map[string]string{}
		for _, name := range required {
			i := index[name]
			if i >= len(record) { return nil, fmt.Errorf("%s:%d: missing value for %q", path, line, name) }
			row[name] = strings.TrimSpace(record[i])
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func LoadInvoices(path string) ([]Invoice, error) {
	rows, err := readCSV(path, []string{"invoice_id","customer_id","subscription_id","currency","amount_due","amount_paid","status"})
	if err != nil { return nil, err }
	out := make([]Invoice, 0, len(rows))
	for i, r := range rows {
		due, e := parseMoney(r["amount_due"]); if e != nil { return nil, fmt.Errorf("%s row %d amount_due: %w", path, i+2, e) }
		paid, e := parseMoney(r["amount_paid"]); if e != nil { return nil, fmt.Errorf("%s row %d amount_paid: %w", path, i+2, e) }
		currency := strings.ToUpper(r["currency"])
		if len(currency) != 3 { return nil, fmt.Errorf("%s row %d: currency must be a three-letter code", path, i+2) }
		if r["invoice_id"] == "" { return nil, fmt.Errorf("%s row %d: invoice_id is required", path, i+2) }
		out = append(out, Invoice{ID:r["invoice_id"], CustomerID:r["customer_id"], SubscriptionID:r["subscription_id"], Currency:currency, Status:strings.ToLower(r["status"]), AmountDue:due, AmountPaid:paid})
	}
	return out, nil
}

func LoadPayments(path string) ([]Payment, error) {
	rows, err := readCSV(path, []string{"payment_id","invoice_id","currency","amount","status"})
	if err != nil { return nil, err }
	out := make([]Payment, 0, len(rows))
	for i, r := range rows {
		amount, e := parseMoney(r["amount"]); if e != nil { return nil, fmt.Errorf("%s row %d amount: %w", path, i+2, e) }
		currency := strings.ToUpper(r["currency"])
		if len(currency) != 3 { return nil, fmt.Errorf("%s row %d: currency must be a three-letter code", path, i+2) }
		if r["payment_id"] == "" || r["invoice_id"] == "" { return nil, fmt.Errorf("%s row %d: payment_id and invoice_id are required", path, i+2) }
		out = append(out, Payment{ID:r["payment_id"], InvoiceID:r["invoice_id"], Currency:currency, Amount:amount, Status:strings.ToLower(r["status"])})
	}
	return out, nil
}

func successful(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) { case "succeeded","paid","captured": return true }
	return false
}

func Run(invoices []Invoice, payments []Payment) ([]Exception, Summary) {
	s := Summary{InvoiceCount:len(invoices), PaymentCount:len(payments), ByCode:map[string]int{}, Disclaimer:"Exceptions are review leads, not confirmed losses, accounting advice, or recovery guarantees."}
	var out []Exception
	byInvoice := map[string]Invoice{}
	paymentIDs := map[string]bool{}
	invoiceIDs := map[string]bool{}
	for _, inv := range invoices {
		if invoiceIDs[inv.ID] {
			out = append(out, Exception{Severity:"high",Code:"duplicate_invoice_id",InvoiceID:inv.ID,Currency:inv.Currency,Description:"Invoice identifier appears more than once.",Evidence:"Duplicate invoice_id in supplied invoice export."})
			continue
		}
		invoiceIDs[inv.ID] = true
		byInvoice[inv.ID] = inv
	}
	totals := map[string]int64{}
	for _, p := range payments {
		if paymentIDs[p.ID] {
			out = append(out, Exception{Severity:"high",Code:"duplicate_payment_id",PaymentID:p.ID,Currency:p.Currency,AmountMinor:p.Amount,Description:"Payment identifier appears more than once.",Evidence:"Duplicate payment_id in supplied payment export."})
			continue
		}
		paymentIDs[p.ID] = true
		inv, ok := byInvoice[p.InvoiceID]
		if !ok {
			out = append(out, Exception{Severity:"high",Code:"orphan_payment",PaymentID:p.ID,Currency:p.Currency,AmountMinor:p.Amount,Description:"Payment references an invoice absent from the supplied invoice export.",Evidence:"payment.invoice_id="+p.InvoiceID})
			continue
		}
		if p.Currency != inv.Currency {
			out = append(out, Exception{Severity:"high",Code:"currency_mismatch",InvoiceID:inv.ID,PaymentID:p.ID,Currency:p.Currency,AmountMinor:p.Amount,Description:"Payment and invoice currencies differ; amounts were not compared.",Evidence:"invoice.currency="+inv.Currency+"; payment.currency="+p.Currency})
			continue
		}
		if successful(p.Status) { totals[inv.ID] += p.Amount }
	}
	for _, inv := range invoices {
		if _, ok := byInvoice[inv.ID]; !ok { continue }
		if inv.AmountPaid > inv.AmountDue {
			out = append(out, Exception{Severity:"high",Code:"paid_exceeds_due",InvoiceID:inv.ID,Currency:inv.Currency,AmountMinor:inv.AmountPaid-inv.AmountDue,Description:"Reported amount paid exceeds invoice amount due.",Evidence:fmt.Sprintf("amount_due_minor=%d; amount_paid_minor=%d",inv.AmountDue,inv.AmountPaid)})
		}
		if totals[inv.ID] != inv.AmountPaid {
			delta := totals[inv.ID]-inv.AmountPaid
			out = append(out, Exception{Severity:"medium",Code:"payment_total_mismatch",InvoiceID:inv.ID,Currency:inv.Currency,AmountMinor:delta,Description:"Sum of successful linked payments differs from invoice amount_paid.",Evidence:fmt.Sprintf("invoice.amount_paid_minor=%d; successful_payments_minor=%d",inv.AmountPaid,totals[inv.ID])})
		}
		if inv.Status == "paid" && inv.AmountPaid < inv.AmountDue {
			out = append(out, Exception{Severity:"high",Code:"paid_status_underpaid",InvoiceID:inv.ID,Currency:inv.Currency,AmountMinor:inv.AmountDue-inv.AmountPaid,Description:"Invoice is marked paid while amount_paid is below amount_due.",Evidence:fmt.Sprintf("status=paid; amount_due_minor=%d; amount_paid_minor=%d",inv.AmountDue,inv.AmountPaid)})
		}
	}
	for _, e := range out { s.ByCode[e.Code]++ }
	s.ExceptionCount = len(out)
	return out, s
}
