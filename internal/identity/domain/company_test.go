package identity

import (
	"strings"
	"testing"
	"time"
)

func TestNewCompany_Success(t *testing.T) {
	createdAt := time.Date(2024, time.January, 10, 12, 0, 0, 0, time.UTC)

	company, err := NewCompany(" company-1 ", " Acme Fish ", "7707083893", "1027700132195", createdAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if company.ID() != "company-1" {
		t.Fatalf("expected trimmed id, got %q", company.ID())
	}
	if company.Name() != "Acme Fish" {
		t.Fatalf("expected trimmed name, got %q", company.Name())
	}
	if company.INN() != "7707083893" {
		t.Fatalf("expected inn to be stored, got %q", company.INN())
	}
	if company.OGRN() != "1027700132195" {
		t.Fatalf("expected ogrn to be stored, got %q", company.OGRN())
	}
	if company.Status() != CompanyStatusActive {
		t.Fatalf("expected initial status %q, got %q", CompanyStatusActive, company.Status())
	}
	if !company.CreatedAt().Equal(createdAt) {
		t.Fatalf("expected createdAt to match input")
	}
}

func TestNewCompany_Validation(t *testing.T) {
	validCreatedAt := time.Now()
	tooLongName := strings.Repeat("a", companyNameMaxLength+1)

	tests := []struct {
		name        string
		companyID   string
		companyName string
		inn         string
		ogrn        string
		createdAt   time.Time
		wantErr     error
	}{
		{
			name:        "empty company id",
			companyID:   " ",
			companyName: "Acme Fish",
			inn:         "7707083893",
			ogrn:        "1027700132195",
			createdAt:   validCreatedAt,
			wantErr:     ErrEmptyCompanyID,
		},
		{
			name:        "empty name",
			companyID:   "company-1",
			companyName: " ",
			inn:         "7707083893",
			ogrn:        "1027700132195",
			createdAt:   validCreatedAt,
			wantErr:     ErrEmptyCompanyName,
		},
		{
			name:        "too long name",
			companyID:   "company-1",
			companyName: tooLongName,
			inn:         "7707083893",
			ogrn:        "1027700132195",
			createdAt:   validCreatedAt,
			wantErr:     ErrCompanyNameTooLong,
		},
		{
			name:        "invalid inn",
			companyID:   "company-1",
			companyName: "Acme Fish",
			inn:         "7707083894",
			ogrn:        "1027700132195",
			createdAt:   validCreatedAt,
			wantErr:     ErrInvalidINN,
		},
		{
			name:        "invalid ogrn",
			companyID:   "company-1",
			companyName: "Acme Fish",
			inn:         "7707083893",
			ogrn:        "1027700132194",
			createdAt:   validCreatedAt,
			wantErr:     ErrInvalidOGRN,
		},
		{
			name:        "zero created at",
			companyID:   "company-1",
			companyName: "Acme Fish",
			inn:         "7707083893",
			ogrn:        "1027700132195",
			createdAt:   time.Time{},
			wantErr:     ErrEmptyCompanyCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCompany(tt.companyID, tt.companyName, tt.inn, tt.ogrn, tt.createdAt)
			if err != tt.wantErr {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCompany_RenameSuccess(t *testing.T) {
	company := mustNewCompany(t)

	if err := company.Rename(" New Name "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if company.Name() != "New Name" {
		t.Fatalf("expected renamed company, got %q", company.Name())
	}
}

func TestCompany_RenameFail(t *testing.T) {
	company := mustNewCompany(t)

	if err := company.Rename(" "); err != ErrEmptyCompanyName {
		t.Fatalf("expected %v, got %v", ErrEmptyCompanyName, err)
	}
}

func TestCompany_BlockActivateArchive(t *testing.T) {
	company := mustNewCompany(t)

	if err := company.Block(); err != nil {
		t.Fatalf("unexpected block error: %v", err)
	}
	if company.Status() != CompanyStatusBlocked {
		t.Fatalf("expected blocked status, got %q", company.Status())
	}

	if err := company.Activate(); err != nil {
		t.Fatalf("unexpected activate error: %v", err)
	}
	if company.Status() != CompanyStatusActive {
		t.Fatalf("expected active status, got %q", company.Status())
	}

	if err := company.Archive(); err != nil {
		t.Fatalf("unexpected archive error: %v", err)
	}
	if company.Status() != CompanyStatusArchived {
		t.Fatalf("expected archived status, got %q", company.Status())
	}
}

func TestCompany_InvalidStatusTransitions(t *testing.T) {
	t.Run("block twice", func(t *testing.T) {
		company := mustNewCompany(t)

		if err := company.Block(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := company.Block(); err != ErrInvalidCompanyState {
			t.Fatalf("expected %v, got %v", ErrInvalidCompanyState, err)
		}
	})

	t.Run("activate from active", func(t *testing.T) {
		company := mustNewCompany(t)

		if err := company.Activate(); err != ErrInvalidCompanyState {
			t.Fatalf("expected %v, got %v", ErrInvalidCompanyState, err)
		}
	})

	t.Run("rename archived", func(t *testing.T) {
		company := mustNewCompany(t)

		if err := company.Archive(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := company.Rename("New Name"); err != ErrCompanyRenameDenied {
			t.Fatalf("expected %v, got %v", ErrCompanyRenameDenied, err)
		}
	})

	t.Run("activate archived", func(t *testing.T) {
		company := mustNewCompany(t)

		if err := company.Archive(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := company.Activate(); err != ErrInvalidCompanyState {
			t.Fatalf("expected %v, got %v", ErrInvalidCompanyState, err)
		}
	})

	t.Run("archive twice", func(t *testing.T) {
		company := mustNewCompany(t)

		if err := company.Archive(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := company.Archive(); err != ErrInvalidCompanyState {
			t.Fatalf("expected %v, got %v", ErrInvalidCompanyState, err)
		}
	})
}

func mustNewCompany(t *testing.T) *Company {
	t.Helper()

	company, err := NewCompany("company-1", "Acme Fish", "7707083893", "1027700132195", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return company
}
