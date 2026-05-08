package app

import "context"

// CompanyVerificationResult contains optional normalized data from external verification.
type CompanyVerificationResult struct {
	Name string
}

// NoopCompanyVerifier is a stub verifier that always succeeds.
type NoopCompanyVerifier struct{}

func NewNoopCompanyVerifier() NoopCompanyVerifier {
	return NoopCompanyVerifier{}
}

func (NoopCompanyVerifier) VerifyCompany(ctx context.Context, inn string, ogrn string) (CompanyVerificationResult, error) {
	_ = ctx
	_ = inn
	_ = ogrn
	return CompanyVerificationResult{}, nil
}
