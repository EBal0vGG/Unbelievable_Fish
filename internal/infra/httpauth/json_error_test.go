package httpauth

import (
	"errors"
	"net/http"
	"testing"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

func TestMapIdentityAuthError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		err            error
		wantStatus     int
		wantCodePrefix string
	}{
		{identityauth.ErrMissingAuthorizationHeader, http.StatusUnauthorized, "MISSING_"},
		{identityauth.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{errors.New("unknown"), http.StatusInternalServerError, "INTERNAL_"},
	}
	for _, tc := range cases {
		st, code, _ := MapIdentityAuthError(tc.err)
		if st != tc.wantStatus {
			t.Fatalf("err %v: status %d, want %d", tc.err, st, tc.wantStatus)
		}
		if len(code) < len(tc.wantCodePrefix) || code[:len(tc.wantCodePrefix)] != tc.wantCodePrefix {
			t.Fatalf("err %v: code %q, want prefix %q", tc.err, code, tc.wantCodePrefix)
		}
	}
}
