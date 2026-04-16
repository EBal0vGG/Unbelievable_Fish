package identity

import "testing"

func TestNewUser_Success(t *testing.T) {
	user, err := NewUser(" user-1 ", " company-1 ", " Alice ", Role(" ADMIN "), " Alice@Example.com ", " hash ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID() != "user-1" {
		t.Fatalf("expected trimmed id, got %q", user.ID())
	}
	if user.CompanyID() != "company-1" {
		t.Fatalf("expected trimmed company id, got %q", user.CompanyID())
	}
	if user.Name() != "Alice" {
		t.Fatalf("expected trimmed name, got %q", user.Name())
	}
	if user.Role() != RoleAdmin {
		t.Fatalf("expected normalized role %q, got %q", RoleAdmin, user.Role())
	}
	if user.Login() != "alice@example.com" {
		t.Fatalf("expected normalized login, got %q", user.Login())
	}
	if user.PasswordHash() != "hash" {
		t.Fatalf("expected trimmed password hash, got %q", user.PasswordHash())
	}
}

func TestNewUser_Validation(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		companyID    string
		userName     string
		role         Role
		login        string
		passwordHash string
		wantErr      error
	}{
		{
			name:         "empty user id",
			userID:       " ",
			companyID:    "company-1",
			userName:     "Alice",
			role:         RoleAdmin,
			login:        "alice@example.com",
			passwordHash: "hash",
			wantErr:      ErrEmptyUserID,
		},
		{
			name:         "empty company id",
			userID:       "user-1",
			companyID:    " ",
			userName:     "Alice",
			role:         RoleAdmin,
			login:        "alice@example.com",
			passwordHash: "hash",
			wantErr:      ErrEmptyCompanyID,
		},
		{
			name:         "empty name",
			userID:       "user-1",
			companyID:    "company-1",
			userName:     " ",
			role:         RoleAdmin,
			login:        "alice@example.com",
			passwordHash: "hash",
			wantErr:      ErrEmptyUserName,
		},
		{
			name:         "empty login",
			userID:       "user-1",
			companyID:    "company-1",
			userName:     "Alice",
			role:         RoleAdmin,
			login:        " ",
			passwordHash: "hash",
			wantErr:      ErrEmptyLogin,
		},
		{
			name:         "invalid email",
			userID:       "user-1",
			companyID:    "company-1",
			userName:     "Alice",
			role:         RoleAdmin,
			login:        "alice-at-example.com",
			passwordHash: "hash",
			wantErr:      ErrInvalidLogin,
		},
		{
			name:         "invalid role",
			userID:       "user-1",
			companyID:    "company-1",
			userName:     "Alice",
			role:         Role("guest"),
			login:        "alice@example.com",
			passwordHash: "hash",
			wantErr:      ErrInvalidRole,
		},
		{
			name:         "empty password hash",
			userID:       "user-1",
			companyID:    "company-1",
			userName:     "Alice",
			role:         RoleAdmin,
			login:        "alice@example.com",
			passwordHash: " ",
			wantErr:      ErrEmptyPasswordHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewUser(tt.userID, tt.companyID, tt.userName, tt.role, tt.login, tt.passwordHash)
			if err != tt.wantErr {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
