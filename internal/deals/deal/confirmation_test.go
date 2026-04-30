package deal

import (
	"testing"
	"time"
)

func TestRequestConfirmationCreatesPending(t *testing.T) {
	item := createPendingDealForConfirmationTest(t)

	confirmation, events, err := item.RequestConfirmation(
		DealConfirmationStageConfirmed,
		item.SupplierID(),
		"user-seller",
		VerificationMethodManual,
		"",
		"",
		"ready",
		nil,
	)
	if err != nil {
		t.Fatalf("request confirmation error: %v", err)
	}
	if confirmation.Status() != DealConfirmationStatusPending {
		t.Fatalf("expected pending confirmation, got %s", confirmation.Status())
	}
	if confirmation.CounterpartyCompanyID() != item.CustomerID() {
		t.Fatalf("expected counterparty %s, got %s", item.CustomerID(), confirmation.CounterpartyCompanyID())
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
}

func TestApplyApprovedConfirmationChangesDealStatus(t *testing.T) {
	item := createPendingDealForConfirmationTest(t)
	confirmation, _, err := item.RequestConfirmation(
		DealConfirmationStageConfirmed,
		item.SupplierID(),
		"user-seller",
		VerificationMethodManual,
		"",
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("request confirmation error: %v", err)
	}
	if _, err := confirmation.Approve(item.CustomerID(), "buyer-user", confirmation.RequestedAt()); err != nil {
		t.Fatalf("approve error: %v", err)
	}

	if _, err := item.ApplyApprovedConfirmation(confirmation); err != nil {
		t.Fatalf("apply confirmation error: %v", err)
	}
	if item.Status() != DealStatusConfirmed {
		t.Fatalf("expected confirmed status, got %s", item.Status())
	}
}

func TestSellerCannotApproveOwnConfirmation(t *testing.T) {
	item := createPendingDealForConfirmationTest(t)
	confirmation, _, err := item.RequestConfirmation(
		DealConfirmationStageConfirmed,
		item.SupplierID(),
		"user-seller",
		VerificationMethodManual,
		"",
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("request confirmation error: %v", err)
	}

	if _, err := confirmation.Approve(item.SupplierID(), "user-seller", confirmation.RequestedAt()); err != ErrCounterpartyRequired {
		t.Fatalf("expected ErrCounterpartyRequired, got %v", err)
	}
}

func TestUnrelatedCompanyCannotRejectConfirmation(t *testing.T) {
	item := createPendingDealForConfirmationTest(t)
	confirmation, _, err := item.RequestConfirmation(
		DealConfirmationStageConfirmed,
		item.SupplierID(),
		"user-seller",
		VerificationMethodManual,
		"",
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("request confirmation error: %v", err)
	}

	if _, err := confirmation.Reject("other-company", "other-user", "no", confirmation.RequestedAt()); err != ErrNotDealParticipant {
		t.Fatalf("expected ErrNotDealParticipant, got %v", err)
	}
}

func TestRejectDoesNotChangeDealStatus(t *testing.T) {
	item := createPendingDealForConfirmationTest(t)
	confirmation, _, err := item.RequestConfirmation(
		DealConfirmationStageConfirmed,
		item.SupplierID(),
		"user-seller",
		VerificationMethodManual,
		"",
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("request confirmation error: %v", err)
	}
	if _, err := confirmation.Reject(item.CustomerID(), "buyer-user", "not yet", confirmation.RequestedAt()); err != nil {
		t.Fatalf("reject error: %v", err)
	}
	if item.Status() != DealStatusPending {
		t.Fatalf("expected pending status, got %s", item.Status())
	}
}

func TestCannotSkipStagesWhenRequestingConfirmation(t *testing.T) {
	item := createPendingDealForConfirmationTest(t)

	_, _, err := item.RequestConfirmation(
		DealConfirmationStageCompleted,
		item.SupplierID(),
		"user-seller",
		VerificationMethodManual,
		"",
		"",
		"",
		nil,
	)
	if err != ErrInvalidStageTransition {
		t.Fatalf("expected ErrInvalidStageTransition, got %v", err)
	}
}

func createPendingDealForConfirmationTest(t *testing.T) *Deal {
	t.Helper()
	factory := NewFactory()
	now := time.Now().UTC()
	projection := NewDealProjection(
		"auc-confirm",
		"seller-1",
		ProductSnapshot{ProductID: "prod-1", Name: "Fish"},
		100,
		now.Add(-time.Hour),
	)
	item, _, err := factory.CreateFromProjection(projection, "buyer-1", 120, now)
	if err != nil {
		t.Fatalf("factory error: %v", err)
	}
	return item
}
