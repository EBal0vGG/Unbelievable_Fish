package wallet

import "testing"

func TestAccountReserveReleaseCapture(t *testing.T) {
	a, err := NewAccount("acc-1", "co-1", CurrencyRUB)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Deposit(10_000); err != nil {
		t.Fatal(err)
	}
	if err := a.Reserve(5_000); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if a.Available() != 5_000 || a.Held() != 5_000 {
		t.Fatalf("after reserve: avail=%d held=%d", a.Available(), a.Held())
	}
	if err := a.Reserve(6_000); err != ErrInsufficientFunds {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
	if err := a.Release(2_000); err != nil {
		t.Fatal(err)
	}
	if a.Available() != 7_000 || a.Held() != 3_000 {
		t.Fatalf("after release: avail=%d held=%d", a.Available(), a.Held())
	}
	if err := a.Capture(3_000); err != nil {
		t.Fatal(err)
	}
	if a.Available() != 7_000 || a.Held() != 0 {
		t.Fatalf("after capture: avail=%d held=%d", a.Available(), a.Held())
	}
	if err := a.Release(1); err != ErrInsufficientHeld {
		t.Fatalf("expected ErrInsufficientHeld, got %v", err)
	}
}

func TestAccountCaptureInsufficientHeld(t *testing.T) {
	a, _ := NewAccount("a", "c", CurrencyRUB)
	_ = a.Deposit(100)
	_ = a.Reserve(50)
	if err := a.Capture(51); err != ErrInsufficientHeld {
		t.Fatalf("got %v", err)
	}
}
