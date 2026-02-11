package deal

import (
	"testing"
	"time"
)

func createTestDeal(t *testing.T) *Deal {
	t.Helper()

	snapshot := ProductSnapshot{
		ProductID: "prod_123",
		Name:      "Test Product",
		Category:  "Electronics",
	}

	deal := &Deal{
		id:              "deal_123",
		customerID:      "winner_456",
		supplierID:      "seller_789",
		auctionID:       "auc_123",
		quantity:        1,
		unitPrice:       1000,
		status:          DealStatusPending,
		typeName:        DealTypeAuction,
		createdAt:       time.Now(),
		productSnapshot: snapshot,
	}

	return deal
}

func TestDeal_Confirm(t *testing.T) {
	deal := createTestDeal(t)

	events, err := deal.Confirm()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deal.Status() != DealStatusConfirmed {
		t.Errorf("expected status confirmed, got %s", deal.Status())
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	_, ok := events[0].(DealConfirmed)
	if !ok {
		t.Errorf("expected DealConfirmed event, got %T", events[0])
	}
}

func TestDeal_PrepareAndSignContract(t *testing.T) {
	deal := createTestDeal(t)
	deal.Confirm()

	// Prepare
	_, err := deal.PrepareContract("CNT-001", "url")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deal.Status() != DealStatusContractPrepared {
		t.Errorf("expected status contract prepared, got %s", deal.Status())
	}

	// Sign
	_, err = deal.SignContract("buyer", "sig_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deal.Status() != DealStatusContractSigned {
		t.Errorf("expected status contract signed, got %s", deal.Status())
	}
	if !deal.hasSignedContract() {
		t.Error("contract should be signed")
	}
}

func TestDeal_RequestPaymentAndMarkAsPaid(t *testing.T) {
	deal := createTestDeal(t)
	deal.Confirm()
	deal.PrepareContract("CNT-001", "")
	deal.SignContract("buyer", "sig")

	// Request payment
	_, err := deal.RequestPayment("INV-001", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deal.Status() != DealStatusPaymentRequested {
		t.Errorf("expected status payment requested, got %s", deal.Status())
	}

	// Mark as paid
	_, err = deal.MarkAsPaid("pay_123", "card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deal.Status() != DealStatusPaid {
		t.Errorf("expected status paid, got %s", deal.Status())
	}
}

func TestDeal_Cancel(t *testing.T) {
	deal := createTestDeal(t)

	events, err := deal.Cancel("buyer changed mind", "customer")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deal.Status() != DealStatusCancelled {
		t.Errorf("expected status cancelled, got %s", deal.Status())
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	event, ok := events[0].(DealCancelled)
	if !ok {
		t.Errorf("expected DealCancelled event, got %T", events[0])
	}
	if event.Reason != "buyer changed mind" {
		t.Errorf("expected reason 'buyer changed mind', got '%s'", event.Reason)
	}
}
