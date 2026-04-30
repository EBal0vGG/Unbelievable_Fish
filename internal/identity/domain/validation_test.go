package identity

import "testing"

func TestValidationHelpers(t *testing.T) {
	if !isBlank(" ") {
		t.Fatalf("expected blank string to be blank")
	}
	if isBlank("value") {
		t.Fatalf("expected non-blank string to be non-blank")
	}

	if !isValidEmail(" User.Name+tag@Example.COM ") {
		t.Fatalf("expected valid email to pass")
	}
	if isValidEmail("invalid-email") {
		t.Fatalf("expected invalid email to fail")
	}

	if !isValidINN("7707083893") {
		t.Fatalf("expected valid INN to pass")
	}
	if isValidINN("7707083894") {
		t.Fatalf("expected invalid INN to fail")
	}

	if !isValidOGRN("1027700132195") {
		t.Fatalf("expected valid OGRN to pass")
	}
	if isValidOGRN("1027700132194") {
		t.Fatalf("expected invalid OGRN to fail")
	}
}
