package catalog

import "testing"

func TestProductPublishRequiresAllFields(t *testing.T) {
	product, _, err := NewProduct("prod-1", "seller-1", "", "", ProcessingFrozen, PackagingBox, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := product.Publish()
	if err != ErrPublishingRuleViolation {
		t.Fatalf("expected ErrPublishingRuleViolation, got %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events on failed publish")
	}
	if product.Status() != ProductStatusDraft {
		t.Fatalf("expected status to remain draft, got %s", product.Status())
	}
}

func TestProductInvalidStateTransition(t *testing.T) {
	product, _, err := NewProduct("prod-2", "seller-2", "Atlantic Salmon", "Salmo salar", ProcessingFrozen, PackagingBox, "10kg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = product.Unpublish()
	if err != ErrForbiddenStateTransition {
		t.Fatalf("expected ErrForbiddenStateTransition, got %v", err)
	}
}
