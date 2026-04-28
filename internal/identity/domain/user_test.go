package identity

import (
	"testing"
	"time"
)

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

func TestNewUser_WithoutCompanyID(t *testing.T) {
	user, err := NewUser("user-1", " ", "Alice", RoleAdmin, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.CompanyID() != "" {
		t.Fatalf("expected empty company id, got %q", user.CompanyID())
	}
}

func TestUserAcceptTerms(t *testing.T) {
	user, err := NewUser("user-1", "company-1", "Alice", RoleAdmin, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	acceptedAt := time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)
	if err := user.AcceptTerms(" v1 ", acceptedAt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.TermsVersion() != "v1" {
		t.Fatalf("expected terms version v1, got %q", user.TermsVersion())
	}
	if !user.TermsAcceptedAt().Equal(acceptedAt) {
		t.Fatalf("expected accepted at %v, got %v", acceptedAt, user.TermsAcceptedAt())
	}
}

func TestUserAcceptTermsValidation(t *testing.T) {
	user, err := NewUser("user-1", "company-1", "Alice", RoleAdmin, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testCases := []struct {
		name       string
		version    string
		acceptedAt time.Time
		wantErr    error
	}{
		{
			name:       "empty version",
			version:    " ",
			acceptedAt: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC),
			wantErr:    ErrEmptyTermsVersion,
		},
		{
			name:       "empty accepted at",
			version:    "v1",
			acceptedAt: time.Time{},
			wantErr:    ErrEmptyTermsAcceptedAt,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := user.AcceptTerms(tc.version, tc.acceptedAt)
			if err != tc.wantErr {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}
