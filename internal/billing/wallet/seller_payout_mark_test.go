package wallet

import (
	"errors"
	"testing"
	"time"
)

func TestSellerPayout_MarkReady(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	p, err := NewSellerPayout("p1", "d1", "i1", "a1", "s1", "b1", 100, CurrencyRUB, SellerPayoutPending, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.MarkReady(now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if p.Status != SellerPayoutReady || p.ReadyAt == nil {
		t.Fatalf("want READY + ready_at, got %+v", p)
	}
	if err := p.MarkReady(now.Add(2 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if p.Status != SellerPayoutReady {
		t.Fatal("second MarkReady should no-op")
	}
}

func TestSellerPayout_MarkReady_rejectsNonPending(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	p, err := NewSellerPayout("p1", "d1", "i1", "a1", "s1", "b1", 100, CurrencyRUB, SellerPayoutCancelled, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.MarkReady(now); !errors.Is(err, ErrSellerPayoutWrongStatus) {
		t.Fatalf("want ErrSellerPayoutWrongStatus got %v", err)
	}
}

func TestSellerPayout_MarkPaid(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	p, err := NewSellerPayout("p1", "d1", "i1", "a1", "s1", "b1", 100, CurrencyRUB, SellerPayoutPending, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.MarkReady(now); err != nil {
		t.Fatal(err)
	}
	if err := p.MarkPaid(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if p.Status != SellerPayoutPaid || p.PaidAt == nil {
		t.Fatalf("want PAID + paid_at, got %+v", p)
	}
	if err := p.MarkPaid(now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func TestSellerPayout_MarkPaid_rejectsFromPending(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	p, err := NewSellerPayout("p1", "d1", "i1", "a1", "s1", "b1", 100, CurrencyRUB, SellerPayoutPending, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.MarkPaid(now); !errors.Is(err, ErrSellerPayoutWrongStatus) {
		t.Fatalf("want ErrSellerPayoutWrongStatus got %v", err)
	}
}
