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
