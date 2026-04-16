package identity

import (
	"strings"
	"time"
)

const companyNameMaxLength = 255

type CompanyStatus string

const (
	CompanyStatusActive   CompanyStatus = "active"
	CompanyStatusBlocked  CompanyStatus = "blocked"
	CompanyStatusArchived CompanyStatus = "archived"
)

type Company struct {
	id        string
	name      string
	inn       string
	ogrn      string
	status    CompanyStatus
	createdAt time.Time
}

func NewCompany(companyID string, name string, inn string, ogrn string, createdAt time.Time) (*Company, error) {
	companyID = strings.TrimSpace(companyID)
	name = strings.TrimSpace(name)
	inn = strings.TrimSpace(inn)
	ogrn = strings.TrimSpace(ogrn)

	if isBlank(companyID) {
		return nil, ErrEmptyCompanyID
	}
	if isBlank(name) {
		return nil, ErrEmptyCompanyName
	}
	if len([]rune(name)) > companyNameMaxLength {
		return nil, ErrCompanyNameTooLong
	}
	if !isValidINN(inn) {
		return nil, ErrInvalidINN
	}
	if !isValidOGRN(ogrn) {
		return nil, ErrInvalidOGRN
	}
	if createdAt.IsZero() {
		return nil, ErrEmptyCompanyCreated
	}

	return &Company{
		id:        companyID,
		name:      name,
		inn:       inn,
		ogrn:      ogrn,
		status:    CompanyStatusActive,
		createdAt: createdAt,
	}, nil
}

func (c *Company) ID() string {
	return c.id
}

func (c *Company) Name() string {
	return c.name
}

func (c *Company) INN() string {
	return c.inn
}

func (c *Company) OGRN() string {
	return c.ogrn
}

func (c *Company) Status() CompanyStatus {
	return c.status
}

func (c *Company) CreatedAt() time.Time {
	return c.createdAt
}

func (c *Company) Rename(name string) error {
	name = strings.TrimSpace(name)

	if c.status == CompanyStatusArchived {
		return ErrCompanyRenameDenied
	}
	if isBlank(name) {
		return ErrEmptyCompanyName
	}
	if len([]rune(name)) > companyNameMaxLength {
		return ErrCompanyNameTooLong
	}

	c.name = name
	return nil
}

func (c *Company) Block() error {
	if c.status != CompanyStatusActive {
		return ErrInvalidCompanyState
	}

	c.status = CompanyStatusBlocked
	return nil
}

func (c *Company) Activate() error {
	if c.status != CompanyStatusBlocked {
		return ErrInvalidCompanyState
	}

	c.status = CompanyStatusActive
	return nil
}

func (c *Company) Archive() error {
	if c.status == CompanyStatusArchived {
		return ErrInvalidCompanyState
	}

	c.status = CompanyStatusArchived
	return nil
}
