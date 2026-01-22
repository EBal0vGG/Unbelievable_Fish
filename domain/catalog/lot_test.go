package catalog

import "testing"

func TestLotPublishRequiresProductPublished(t *testing.T) {
	lot, _, err := NewLot("lot-1", "prod-1", "seller-1", 100, UnitKg, "FOB", "-18C")
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

func TestLotQuantityValidation(t *testing.T) {
	_, _, err := NewLot("lot-2", "prod-2", "seller-2", 0, UnitKg, "CIF", "-18C")
	if err != ErrInvalidQuantity {
		t.Fatalf("expected ErrInvalidQuantity, got %v", err)
	}
}

func TestLotSoldRules(t *testing.T) {
	lot, _, err := NewLot("lot-3", "prod-3", "seller-3", 50, UnitTon, "EXW", "-10C")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.Publish(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = lot.MarkSold("")
	if err != ErrInvalidIdentifier {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}
	if lot.Status() != LotStatusPublished {
		t.Fatalf("expected status to remain published, got %s", lot.Status())
	}

	_, err = lot.MarkSold("deal-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lot.Status() != LotStatusSold {
		t.Fatalf("expected status to be sold, got %s", lot.Status())
	}
	if lot.DealID() != "deal-1" {
		t.Fatalf("expected deal id to be set")
	}

	_, err = lot.Unpublish()
	if err != ErrForbiddenStateTransition {
		t.Fatalf("expected ErrForbiddenStateTransition, got %v", err)
	}
}
