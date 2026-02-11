package catalog

import "testing"

func TestNewProduct_Validation(t *testing.T) {
	var pt ProcessingType

	_, _, err := NewProduct("", "fish-1", 1000, "KG", "L", pt)
	if err != ErrInvalidIdentifier {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}

	_, _, err = NewProduct("prod-1", "", 1000, "KG", "L", pt)
	if err != ErrInvalidIdentifier {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}

	_, _, err = NewProduct("prod-1", "fish-1", 0, "KG", "L", pt)
	if err != ErrInvalidWeight {
		t.Fatalf("expected ErrInvalidWeight, got %v", err)
	}

	_, _, err = NewProduct("prod-1", "fish-1", 1000, "", "L", pt)
	if err != ErrInvalidIdentifier {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestProduct_PublishAndUnpublish(t *testing.T) {
	var pt ProcessingType

	p, events, err := NewProduct("prod-1", "fish-1", 1000, "KG", "L", pt)
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
	var pt ProcessingType

	p, _, err := NewProduct("prod-2", "fish-2", 2000, "KG", "M", pt)
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
