package catalog

import (
	"testing"
	"time"
)

func newSchedule() *AuctionSchedule {
	return &AuctionSchedule{startsAt: NewInstant(time.Now().Add(time.Hour))}
}

func TestLotPublishRequiresProductPublished(t *testing.T) {
	lot, _, err := NewLot("lot-1", "prod-1", "seller-1", int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.AssignAuctionID("auc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(false)
	if err != ErrPublishingRuleViolation {
		t.Fatalf("expected ErrPublishingRuleViolation, got %v", err)
	}
	if lot.Status() != LotStatusDraft {
		t.Fatalf("expected status to remain draft, got %s", lot.Status())
	}
}

func TestLotPublishRequiresAuctionID(t *testing.T) {
	lot, _, err := NewLot("lot-0", "prod-0", "seller-0", int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(true)
	if err != ErrAuctionIDRequired {
		t.Fatalf("expected ErrAuctionIDRequired, got %v", err)
	}
	if lot.Status() != LotStatusDraft {
		t.Fatalf("expected status to remain draft, got %s", lot.Status())
	}
}

func TestLotStartPriceValidation(t *testing.T) {
	_, _, err := NewLot("lot-2", "prod-2", "seller-2", int64(0), newSchedule())
	if err != ErrInvalidPrice {
		t.Fatalf("expected ErrInvalidPrice, got %v", err)
	}
}

func TestAssignAuctionIDCannotReassign(t *testing.T) {
	lot, _, err := NewLot("lot-4", "prod-4", "seller-4", int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.AssignAuctionID("auc-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.AssignAuctionID("auc-4b")
	if err != ErrAlreadyAssigned {
		t.Fatalf("expected ErrAlreadyAssigned, got %v", err)
	}
}

func TestUnpublishFromPublished(t *testing.T) {
	lot, _, err := NewLot("lot-5", "prod-5", "seller-5", int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.AssignAuctionID("auc-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Unpublish()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lot.Status() != LotStatusCancelled {
		t.Fatalf("expected status to be cancelled, got %s", lot.Status())
	}
}
