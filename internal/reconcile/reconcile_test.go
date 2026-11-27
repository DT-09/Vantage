package reconcile

import "testing"

func TestRunFindsPaymentMismatchAndOrphan(t *testing.T) {
	invoices := []Invoice{{ID:"inv_1",Currency:"USD",Status:"paid",AmountDue:10000,AmountPaid:10000}}
	payments := []Payment{
		{ID:"pay_1",InvoiceID:"inv_1",Currency:"USD",Amount:9000,Status:"succeeded"},
		{ID:"pay_2",InvoiceID:"missing",Currency:"USD",Amount:500,Status:"succeeded"},
	}
	got, summary := Run(invoices,payments)
	if summary.ExceptionCount != 2 { t.Fatalf("expected 2 exceptions, got %d: %#v",summary.ExceptionCount,got) }
	if summary.ByCode["payment_total_mismatch"] != 1 || summary.ByCode["orphan_payment"] != 1 { t.Fatalf("unexpected exception codes: %#v",summary.ByCode) }
}

func TestRunDoesNotTreatPendingPaymentsAsCollected(t *testing.T) {
	invoices := []Invoice{{ID:"inv_1",Currency:"EUR",Status:"open",AmountDue:1000,AmountPaid:0}}
	payments := []Payment{{ID:"pay_1",InvoiceID:"inv_1",Currency:"EUR",Amount:1000,Status:"pending"}}
	got, _ := Run(invoices,payments)
	if len(got) != 0 { t.Fatalf("pending payment must not count as collected: %#v",got) }
}

func TestParseMoney(t *testing.T) {
	cases := map[string]int64{"0":0,"12":1200,"12.3":1230,"12.34":1234}
	for input, want := range cases { got,err:=parseMoney(input); if err!=nil || got!=want { t.Fatalf("parseMoney(%q)=(%d,%v), want %d",input,got,err,want) } }
	if _,err:=parseMoney("1.234"); err==nil { t.Fatal("expected rejection of more than two decimal places") }
}
