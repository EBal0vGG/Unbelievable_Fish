package deal

import (
	"errors"
	"strconv"
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
	logTest(t)

	deal := createTestDeal(t)
	logMsg(t, "deal id="+deal.ID()+" status="+string(deal.Status()))

	events, err := deal.Confirm()

	if err != nil {
		logMsg(t, "confirm error="+err.Error())
		t.Fatalf("unexpected error: %v", err)
	}
	logMsg(t, "confirm events="+strconv.Itoa(len(events))+" error=<nil>")

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
	logTest(t)

	deal := createTestDeal(t)
	logMsg(t, "deal id="+deal.ID()+" status="+string(deal.Status()))

	deal.Confirm()
	logMsg(t, "deal confirmed status="+string(deal.Status()))

	// Prepare
	_, err := deal.PrepareContract("CNT-001", "url")
	if err != nil {
		logMsg(t, "prepare contract error="+err.Error())
		t.Fatalf("unexpected error: %v", err)
	}
	logMsg(t, "contract prepared number=CNT-001 status="+string(deal.Status()))

	if deal.Status() != DealStatusContractPrepared {
		t.Errorf("expected status contract prepared, got %s", deal.Status())
	}

	// Sign
	_, err = deal.SignContract("buyer", "sig_123")
	if err != nil {
		logMsg(t, "sign contract error="+err.Error())
		t.Fatalf("unexpected error: %v", err)
	}
	logMsg(t, "contract signed by=buyer status="+string(deal.Status()))

	if deal.Status() != DealStatusContractSigned {
		t.Errorf("expected status contract signed, got %s", deal.Status())
	}
	if !deal.hasSignedContract() {
		t.Error("contract should be signed")
	}
}

func TestDeal_RequestPaymentAndMarkAsPaid(t *testing.T) {
	logTest(t)

	deal := createTestDeal(t)
	logMsg(t, "deal id="+deal.ID()+" total="+strconv.FormatInt(deal.CalculateTotal(), 10))

	deal.Confirm()
	deal.PrepareContract("CNT-001", "")
	deal.SignContract("buyer", "sig")
	logMsg(t, "contract signed status="+string(deal.Status()))

	// Request payment
	_, err := deal.RequestPayment("INV-001", nil)
	if err != nil {
		logMsg(t, "request payment error="+err.Error())
		t.Fatalf("unexpected error: %v", err)
	}
	logMsg(t, "payment requested invoice=INV-001 amount="+strconv.FormatInt(deal.CalculateTotal(), 10))

	if deal.Status() != DealStatusPaymentRequested {
		t.Errorf("expected status payment requested, got %s", deal.Status())
	}

	// Mark as paid
	_, err = deal.MarkAsPaid("pay_123", "card")
	if err != nil {
		logMsg(t, "mark as paid error="+err.Error())
		t.Fatalf("unexpected error: %v", err)
	}
	logMsg(t, "deal paid payment_id=pay_123 type=card")

	if deal.Status() != DealStatusPaid {
		t.Errorf("expected status paid, got %s", deal.Status())
	}
}

func TestDeal_Cancel(t *testing.T) {
	logTest(t)

	deal := createTestDeal(t)
	logMsg(t, "deal id="+deal.ID()+" status="+string(deal.Status()))

	events, err := deal.Cancel("buyer changed mind", "customer")

	if err != nil {
		logMsg(t, "cancel error="+err.Error())
		t.Fatalf("unexpected error: %v", err)
	}
	logMsg(t, "deal cancelled reason=buyer changed mind cancelled_by=customer")

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

func TestDeal_Cancel_WinnerRejectEmitsWinnerRejected(t *testing.T) {
	logTest(t)

	d := createTestDeal(t)
	events, err := d.Cancel(DealCancelReasonWinnerRejected, "customer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if _, ok := events[0].(DealCancelled); !ok {
		t.Fatalf("expected DealCancelled first, got %T", events[0])
	}
	wr, ok := events[1].(WinnerRejected)
	if !ok {
		t.Fatalf("expected WinnerRejected second, got %T", events[1])
	}
	if wr.Reason != DealCancelReasonWinnerRejected {
		t.Fatalf("expected normalized reason %q, got %q", DealCancelReasonWinnerRejected, wr.Reason)
	}
	if wr.CompanyID != d.CustomerID() || wr.AuctionID != d.AuctionID() {
		t.Fatalf("unexpected WinnerRejected fields: %+v", wr)
	}
}

func TestDeal_Cancel_LegacyDeadlineReasonNormalizesOnWinnerRejected(t *testing.T) {
	d := createTestDeal(t)
	events, err := d.Cancel(DealCancelReasonLegacyDeadlineExceeded, "system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wr := events[1].(WinnerRejected)
	if wr.Reason != DealCancelReasonConfirmationTimeout {
		t.Fatalf("expected %q, got %q", DealCancelReasonConfirmationTimeout, wr.Reason)
	}
}

func TestDeal_Cancel_WinnerRejectRejectedAfterConfirm(t *testing.T) {
	d := createTestDeal(t)
	if _, err := d.Confirm(); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	_, err := d.Cancel(DealCancelReasonWinnerRejected, "customer")
	if !errors.Is(err, ErrCannotDeclineWinnerAfterConfirm) {
		t.Fatalf("expected ErrCannotDeclineWinnerAfterConfirm, got %v", err)
	}
}
