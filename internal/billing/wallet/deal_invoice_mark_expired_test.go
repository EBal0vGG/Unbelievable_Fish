package wallet

import (
	"errors"
	"testing"
	"time"
)

func TestDealInvoice_MarkExpired_idempotentPaidAndExpired(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	inv, err := NewDealInvoice("i1", "d1", "a1", "b1", "s1", 100, 10, CurrencyRUB, "fake", now.Add(time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.AttachProvider("p1", "http://pay"); err != nil {
		t.Fatal(err)
	}
	if err := inv.MarkExpired(now); err != nil {
		t.Fatal(err)
	}
	if inv.Status != InvoiceExpired || inv.ExpiredAt == nil {
		t.Fatalf("expected EXPIRED with ExpiredAt, got %s %+v", inv.Status, inv.ExpiredAt)
	}
	if err := inv.MarkExpired(now); err != nil {
		t.Fatal(err)
	}

	paidInv, err := NewDealInvoice("i2", "d2", "a1", "b1", "s1", 50, 5, CurrencyRUB, "fake", now.Add(time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := paidInv.AttachProvider("p2", "http://pay"); err != nil {
		t.Fatal(err)
	}
	if err := paidInv.MarkPaid(paidInv.TotalAmount, paidInv.Currency, now); err != nil {
		t.Fatal(err)
	}
	if err := paidInv.MarkExpired(now); err != nil {
		t.Fatal(err)
	}
	if paidInv.Status != InvoicePaid {
		t.Fatalf("PAID invoice should stay PAID after MarkExpired noop, got %s", paidInv.Status)
	}
}

func TestDealInvoice_MarkExpired_wrongStatus(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	inv, err := NewDealInvoice("i1", "d1", "a1", "b1", "s1", 100, 10, CurrencyRUB, "fake", now.Add(time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.MarkExpired(now); !errors.Is(err, ErrInvoiceNotExpirable) {
		t.Fatalf("want ErrInvoiceNotExpirable from PENDING, got %v", err)
	}
}
