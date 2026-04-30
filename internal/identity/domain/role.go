package identity

import "strings"

type Role string

const (
	RoleAdmin       Role = "admin"
	RoleSeller      Role = "seller"
	RoleBuyer       Role = "buyer"
	RoleBuyerSeller Role = "buyer_seller"
)

func IsValidRole(role Role) bool {
	switch normalizeRole(role) {
	case RoleAdmin, RoleSeller, RoleBuyer, RoleBuyerSeller:
		return true
	default:
		return false
	}
}

func IncludesRole(current Role, required Role) bool {
	current = normalizeRole(current)
	required = normalizeRole(required)
	if current == required {
		return true
	}
	return current == RoleBuyerSeller && (required == RoleBuyer || required == RoleSeller)
}

func normalizeRole(role Role) Role {
	return Role(strings.ToLower(strings.TrimSpace(string(role))))
}
