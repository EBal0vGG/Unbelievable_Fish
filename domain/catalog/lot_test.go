package catalog

import (
	"testing"
	"time"
)

func TestLotPublishRequiresProductPublished(t *testing.T) {
	schedule := &AuctionSchedule{startsAt: NewInstant(time.Now().Add(time.Hour))}

	lot, _, err := NewLot("lot-1", "prod-1", "seller-1", int64(100), schedule)
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
	schedule := &AuctionSchedule{startsAt: NewInstant(time.Now().Add(time.Hour))}

	lot, _, err := NewLot("lot-0", "prod-0", "seller-0", int64(100), schedule)
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
	schedule := &AuctionSchedule{startsAt: NewInstant(time.Now().Add(time.Hour))}

	_, _, err := NewLot("lot-2", "prod-2", "seller-2", int64(0), schedule)
	if err != ErrInvalidPrice {
		t.Fatalf("expected ErrInvalidPrice, got %v", err)
	}
}

func TestLotSoldRules(t *testing.T) {
	schedule := &AuctionSchedule{startsAt: NewInstant(time.Now().Add(time.Hour))}

	lot, _, err := NewLot("lot-3", "prod-3", "seller-3", int64(50), schedule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.AssignAuctionID("auc-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.MarkSold("", int64(100))
	if err != ErrInvalidIdentifier {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}
	if lot.Status() != LotStatusPublished {
		t.Fatalf("expected status to remain published, got %s", lot.Status())
	}

	_, err = lot.MarkSold("deal-1", int64(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lot.Status() != LotStatusSold {
		t.Fatalf("expected status to be sold, got %s", lot.Status())
	}
	if lot.DealID() != "deal-1" {
		t.Fatalf("expected deal id to be set, got %q", lot.DealID())
	}

	_, err = lot.Unpublish()
	if err != ErrForbiddenStateTransition {
		t.Fatalf("expected ErrForbiddenStateTransition, got %v", err)
	}
}
