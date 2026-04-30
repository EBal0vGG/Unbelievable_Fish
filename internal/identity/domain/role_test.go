package identity

import "testing"

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{name: "admin", role: RoleAdmin, want: true},
		{name: "seller", role: RoleSeller, want: true},
		{name: "buyer", role: RoleBuyer, want: true},
		{name: "buyer_seller", role: RoleBuyerSeller, want: true},
		{name: "normalized", role: Role(" ADMIN "), want: true},
		{name: "invalid", role: Role("guest"), want: false},
		{name: "empty", role: Role(" "), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidRole(tt.role); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestIncludesRole(t *testing.T) {
	tests := []struct {
		name     string
		current  Role
		required Role
		want     bool
	}{
		{name: "exact seller", current: RoleSeller, required: RoleSeller, want: true},
		{name: "exact buyer", current: RoleBuyer, required: RoleBuyer, want: true},
		{name: "exact admin", current: RoleAdmin, required: RoleAdmin, want: true},
		{name: "buyer_seller includes seller", current: RoleBuyerSeller, required: RoleSeller, want: true},
		{name: "buyer_seller includes buyer", current: RoleBuyerSeller, required: RoleBuyer, want: true},
		{name: "buyer_seller not admin", current: RoleBuyerSeller, required: RoleAdmin, want: false},
		{name: "seller not buyer", current: RoleSeller, required: RoleBuyer, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IncludesRole(tt.current, tt.required); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
