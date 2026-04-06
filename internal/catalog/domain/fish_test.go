package catalog

import "testing"

func TestNewFish_Validation(t *testing.T) {
	_, err := NewFish("", "Cod", "desc")
	if err != ErrInvalidIdentifier {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}

	_, err = NewFish("fish-1", "", "desc")
	if err != ErrInvalidIdentifier {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestFish_Update(t *testing.T) {
	f, err := NewFish("fish-1", "Cod", "old")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = f.Update("NewCod", "new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Name() != "NewCod" {
		t.Fatalf("expected name to update")
	}
	if f.Description() != "new" {
		t.Fatalf("expected description to update")
	}
}
