package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/DT-09/Vantage/internal/reconcile"
)

func main() {
	invoicesPath := flag.String("invoices", "invoices.csv", "invoice CSV export")
	paymentsPath := flag.String("payments", "payments.csv", "payment CSV export")
	outDir := flag.String("out", "vantage-review", "output directory")
	flag.Parse()
	invoices, err := reconcile.LoadInvoices(*invoicesPath); check(err)
	payments, err := reconcile.LoadPayments(*paymentsPath); check(err)
	exceptions, summary := reconcile.Run(invoices, payments)
	check(os.MkdirAll(*outDir, 0700))
	check(writeExceptions(filepath.Join(*outDir, "exceptions.csv"), exceptions))
	data, err := json.MarshalIndent(summary, "", "  "); check(err)
	check(os.WriteFile(filepath.Join(*outDir, "summary.json"), append(data, '\n'), 0600))
	fmt.Printf("Review complete: %d invoices, %d payments, %d exceptions. See %s\n", summary.InvoiceCount, summary.PaymentCount, summary.ExceptionCount, *outDir)
}

func writeExceptions(path string, items []reconcile.Exception) error {
	f, err := os.Create(path); if err != nil { return err }; defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write([]string{"severity","code","invoice_id","payment_id","currency","amount_minor","description","evidence"}); err != nil { return err }
	for _, e := range items {
		if err := w.Write([]string{e.Severity,e.Code,e.InvoiceID,e.PaymentID,e.Currency,strconv.FormatInt(e.AmountMinor,10),e.Description,e.Evidence}); err != nil { return err }
	}
	w.Flush(); return w.Error()
}

func check(err error) { if err != nil { fmt.Fprintln(os.Stderr,"vantage-reconcile:",err); os.Exit(1) } }
