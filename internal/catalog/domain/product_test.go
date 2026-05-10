package catalog

import "testing"

func TestNewProduct_Validation(t *testing.T) {
	pt := ProcessingType("proc-1")

	_, _, err := NewProduct("", "fish-1", "seller-1", 1000, "KG", "L", pt)
	if err != ErrInvalidIdentifier {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}

	_, _, err = NewProduct("prod-1", "", "seller-1", 1000, "KG", "L", pt)
	if err != ErrInvalidIdentifier {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}

	_, _, err = NewProduct("prod-1", "fish-1", "", 1, "KG", "L", pt)
	if err != ErrInvalidIdentifier {
		t.Fatalf("expected ErrInvalidIdentifier for blank seller, got %v", err)
	}

	_, _, err = NewProduct("prod-1", "fish-1", "seller-1", 0, "KG", "L", pt)
	if err != ErrInvalidWeight {
		t.Fatalf("expected ErrInvalidWeight, got %v", err)
	}

	_, _, err = NewProduct("prod-1", "fish-1", "seller-1", 1, "", "L", pt)
	if err != ErrInvalidUnit {
		t.Fatalf("expected ErrInvalidUnit, got %v", err)
	}

	_, _, err = NewProduct("prod-1", "fish-1", "seller-1", 1, "KG", "L", ProcessingType(""))
	if err != ErrInvalidEnum {
		t.Fatalf("expected ErrInvalidEnum, got %v", err)
	}
}

func TestProduct_PublishAndUnpublish(t *testing.T) {
	pt := ProcessingType("proc-1")

	p, events, err := NewProduct("prod-1", "fish-1", "seller-1", 1000, "KG", "L", pt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if _, ok := events[0].(ProductCreated); !ok {
		t.Fatalf("expected ProductCreated event")
	}

	initial := p.Status()

	evs, err := p.Publish()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status() == initial {
		t.Fatalf("expected status to change after publish")
	}
	if len(evs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evs))
	}
	if _, ok := evs[0].(ProductPublished); !ok {
		t.Fatalf("expected ProductPublished event")
	}

	evs, err = p.Unpublish()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status() != initial {
		t.Fatalf("expected status to return to initial after unpublish")
	}
	if len(evs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evs))
	}
	if _, ok := evs[0].(ProductUnpublished); !ok {
		t.Fatalf("expected ProductUnpublished event")
	}
}

func TestProduct_UpdateOnlyInDraft(t *testing.T) {
	pt := ProcessingType("proc-1")

	p, _, err := NewProduct("prod-2", "fish-2", "seller-1", 2000, "KG", "M", pt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = p.Publish()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = p.Update("fish-x", 123, "KG", "S", pt)
	if err != ErrModificationNotAllowed {
		t.Fatalf("expected ErrModificationNotAllowed, got %v", err)
	}
}

func TestProduct_WeightAndUnitNormalization(t *testing.T) {
	pt := ProcessingType("proc-1")

	p, _, err := NewProduct("prod-3", "fish-3", "seller-1", 123.5, " kg ", "L", pt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Weight() != 123.5 {
		t.Fatalf("expected weight 123.5, got %v", p.Weight())
	}
	if p.Unit() != "kg" {
		t.Fatalf("expected unit kg, got %s", p.Unit())
	}
}
