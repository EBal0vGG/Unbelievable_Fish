package catalog

import (
	"testing"
	"time"
)

func newSchedule() *AuctionSchedule {
	return NewAuctionScheduleAt(time.Now().Add(time.Hour), time.Hour)
}

func newProductSnapshot() ProductSnapshot {
	return ProductSnapshot{
		ProductID:      "prod-1",
		Name:           "Fish",
		Weight:         10,
		Unit:           "kg",
		Size:           "M",
		ProcessingType: ProcessingType("frozen"),
	}
}

func TestLotPublishRequiresProductPublished(t *testing.T) {
	lot, _, err := NewLot("lot-1", "prod-1", "seller-1", "", 10.0, int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.AssignAuctionID("auc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(false, newProductSnapshot())
	if err != ErrPublishingRuleViolation {
		t.Fatalf("expected ErrPublishingRuleViolation, got %v", err)
	}
	if lot.Status() != LotStatusDraft {
		t.Fatalf("expected status to remain draft, got %s", lot.Status())
	}
}

func TestLotPublishAllowsMissingAuctionID(t *testing.T) {
	lot, _, err := NewLot("lot-0", "prod-0", "seller-0", "", 10.0, int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(true, newProductSnapshot())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lot.Status() != LotStatusPublished {
		t.Fatalf("expected status to become published, got %s", lot.Status())
	}
}

func TestLotStartPriceValidation(t *testing.T) {
	_, _, err := NewLot("lot-2", "prod-2", "seller-2", "", 10.0, int64(0), newSchedule())
	if err != ErrInvalidPrice {
		t.Fatalf("expected ErrInvalidPrice, got %v", err)
	}
}

func TestAssignAuctionIDCannotReassign(t *testing.T) {
	lot, _, err := NewLot("lot-4", "prod-4", "seller-4", "", 10.0, int64(100), newSchedule())
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
	lot, _, err := NewLot("lot-5", "prod-5", "seller-5", "", 10.0, int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.AssignAuctionID("auc-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(true, newProductSnapshot())
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

func TestLotCloseFromPublished(t *testing.T) {
	lot, _, err := NewLot("lot-6", "prod-6", "seller-6", "", 10.0, int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.AssignAuctionID("auc-6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(true, newProductSnapshot())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := lot.Close(int64(150))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lot.Status() != LotStatusClosed {
		t.Fatalf("expected status to be closed, got %s", lot.Status())
	}
	if lot.FinalPrice() != int64(150) {
		t.Fatalf("expected final price to be updated, got %d", lot.FinalPrice())
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	closed, ok := events[0].(LotClosed)
	if !ok {
		t.Fatalf("expected LotClosed event")
	}
	if closed.FinalPrice != int64(150) {
		t.Fatalf("expected final price in event, got %d", closed.FinalPrice)
	}
	if closed.Status != LotStatusClosed {
		t.Fatalf("expected closed status in event, got %s", closed.Status)
	}
	if closed.LotID != lot.ID() {
		t.Fatalf("expected lot id in event to match")
	}
}

func TestLotCloseInvalidPrice(t *testing.T) {
	lot, _, err := NewLot("lot-7", "prod-7", "seller-7", "", 10.0, int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.AssignAuctionID("auc-7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(true, newProductSnapshot())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := lot.Close(0)
	if err != ErrInvalidPrice {
		t.Fatalf("expected ErrInvalidPrice, got %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events on failed close")
	}
	if lot.Status() != LotStatusPublished {
		t.Fatalf("expected status to remain published, got %s", lot.Status())
	}
}

func TestLotQuantityAllowsFraction(t *testing.T) {
	lot, _, err := NewLot("lot-8", "prod-8", "seller-8", "", 123.5, int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lot.Quantity() != 123.5 {
		t.Fatalf("expected quantity 123.5, got %v", lot.Quantity())
	}
}

func TestLotPhotoStored(t *testing.T) {
	lot, _, err := NewLot("lot-9", "prod-9", "seller-9", "  https://img  ", 10.0, int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lot.Photo() != "https://img" {
		t.Fatalf("expected trimmed photo, got %q", lot.Photo())
	}
}

func TestLotStartPriceInt64(t *testing.T) {
	_, _, err := NewLot("lot-10", "prod-10", "seller-10", "", 10.0, int64(123), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLotUpdateCurrentPriceEmitsEvent(t *testing.T) {
	lot, _, err := NewLot("lot-11", "prod-11", "seller-11", "", 10.0, int64(100), newSchedule())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.AssignAuctionID("auc-11")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(true, newProductSnapshot())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := lot.UpdateCurrentPrice(int64(135))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lot.CurPrice() != int64(135) {
		t.Fatalf("expected current price to be updated, got %d", lot.CurPrice())
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}

	updated, ok := events[0].(LotPriceUpdated)
	if !ok {
		t.Fatalf("expected LotPriceUpdated event")
	}
	if updated.LotID != lot.ID() {
		t.Fatalf("expected lot id to match")
	}
	if updated.AuctionID != "auc-11" {
		t.Fatalf("expected auction id to match, got %s", updated.AuctionID)
	}
	if updated.CurrentPrice != int64(135) {
		t.Fatalf("expected current price in event, got %d", updated.CurrentPrice)
	}
	if updated.Status != LotStatusPublished {
		t.Fatalf("expected published status in event, got %s", updated.Status)
	}
}
