package httpapi

import (
	"testing"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

func TestMapErrorReturnsDetailedValidationErrors(t *testing.T) {
	testCases := []struct {
		name    string
		err     error
		code    string
		message string
	}{
		{
			name:    "invalid inn",
			err:     identity.ErrInvalidINN,
			code:    "INVALID_INN",
			message: "inn is invalid",
		},
		{
			name:    "invalid ogrn",
			err:     identity.ErrInvalidOGRN,
			code:    "INVALID_OGRN",
			message: "ogrn is invalid",
		},
		{
			name:    "company name required",
			err:     identity.ErrEmptyCompanyName,
			code:    "COMPANY_NAME_REQUIRED",
			message: "company name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapError(tc.err)
			if got.Code != tc.code {
				t.Fatalf("expected code %q, got %q", tc.code, got.Code)
			}
			if got.Message != tc.message {
				t.Fatalf("expected message %q, got %q", tc.message, got.Message)
			}
		})
	}
}

func TestMapErrorReturnsTermsAcceptanceErrors(t *testing.T) {
	testCases := []struct {
		name    string
		err     error
		code    string
		message string
	}{
		{
			name:    "terms acceptance required",
			err:     identityapp.ErrTermsAcceptanceRequired,
			code:    "TERMS_ACCEPTANCE_REQUIRED",
			message: "terms acceptance is required",
		},
		{
			name:    "terms version required",
			err:     identityapp.ErrTermsVersionRequired,
			code:    "TERMS_VERSION_REQUIRED",
			message: "terms version is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapError(tc.err)
			if got.Code != tc.code {
				t.Fatalf("expected code %q, got %q", tc.code, got.Code)
			}
			if got.Message != tc.message {
				t.Fatalf("expected message %q, got %q", tc.message, got.Message)
			}
		})
	}
}
