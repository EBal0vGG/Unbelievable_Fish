package identity

import "strings"

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleSeller Role = "seller"
	RoleBuyer  Role = "buyer"
)

func IsValidRole(role Role) bool {
	switch normalizeRole(role) {
	case RoleAdmin, RoleSeller, RoleBuyer:
		return true
	default:
		return false
	}
}

func normalizeRole(role Role) Role {
	return Role(strings.ToLower(strings.TrimSpace(string(role))))
}
