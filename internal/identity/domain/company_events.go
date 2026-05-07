package identity

// CompanyCreated is emitted when a new company row is persisted (outbox → integration → billing).
type CompanyCreated struct {
	CompanyID string
}
