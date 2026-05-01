package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type RegisterUser struct {
	users     UserRepository
	companies CompanyRepository
	hasher    PasswordHasher
	ids       IDGenerator
	clock     Clock
}

func NewRegisterUser(
	users UserRepository,
	companies CompanyRepository,
	hasher PasswordHasher,
	ids IDGenerator,
	clock Clock,
) (*RegisterUser, error) {
	if users == nil {
		return nil, ErrNilUserRepository
	}
	if companies == nil {
		return nil, ErrNilCompanyRepository
	}
	if hasher == nil {
		return nil, ErrNilPasswordHasher
	}
	if ids == nil {
		ids = NewRandomIDGenerator()
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &RegisterUser{
		users:     users,
		companies: companies,
		hasher:    hasher,
		ids:       ids,
		clock:     clock,
	}, nil
}

func (uc *RegisterUser) Execute(ctx context.Context, cmd RegisterUserCommand) (UserDTO, error) {
	if strings.TrimSpace(cmd.Password) == "" {
		return UserDTO{}, ErrPasswordRequired
	}
	if !cmd.AcceptedTerms {
		return UserDTO{}, ErrTermsAcceptanceRequired
	}
	if strings.TrimSpace(cmd.TermsVersion) == "" {
		return UserDTO{}, ErrTermsVersionRequired
	}
	if cmd.Role == identity.RoleAdmin {
		return UserDTO{}, ErrAdminRegistrationForbidden
	}

	companyID, err := uc.resolveCompanyID(ctx, cmd)
	if err != nil {
		return UserDTO{}, err
	}

	login := strings.ToLower(strings.TrimSpace(cmd.Login))
	existing, err := uc.users.GetByLogin(ctx, login)
	if err == nil && existing != nil {
		return UserDTO{}, ErrLoginAlreadyUsed
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return UserDTO{}, err
	}

	passwordHash, err := uc.hasher.Hash(cmd.Password)
	if err != nil {
		return UserDTO{}, err
	}

	user, err := identity.NewUser(
		uc.ids.NewUserID(),
		companyID,
		cmd.Name,
		cmd.Role,
		login,
		passwordHash,
	)
	if err != nil {
		return UserDTO{}, err
	}
	if err := user.AcceptTerms(cmd.TermsVersion, uc.clock.Now()); err != nil {
		return UserDTO{}, err
	}
	if err := uc.users.Save(ctx, user); err != nil {
		return UserDTO{}, err
	}
	return userDTOFromDomain(user), nil
}

func (uc *RegisterUser) resolveCompanyID(ctx context.Context, cmd RegisterUserCommand) (string, error) {
	companyID := strings.TrimSpace(cmd.CompanyID)
	if companyID != "" {
		if _, err := uc.companies.GetByID(ctx, companyID); err != nil {
			if errors.Is(err, ErrCompanyNotFound) {
				return "", ErrCompanyNotFound
			}
			return "", err
		}
		return companyID, nil
	}

	companyINN := strings.TrimSpace(cmd.CompanyINN)
	companyOGRN := strings.TrimSpace(cmd.CompanyOGRN)
	if companyINN == "" || companyOGRN == "" {
		// No full requisites: create a shell company so the user always has company_id for B2B flows.
		return uc.ensureDummyCompany(ctx, cmd)
	}

	company, err := uc.companies.GetByRequisites(ctx, companyINN, companyOGRN)
	if err != nil {
		if errors.Is(err, ErrCompanyNotFound) {
			return "", ErrCompanyNotFound
		}
		return "", err
	}
	return company.ID(), nil
}

func (uc *RegisterUser) ensureDummyCompany(ctx context.Context, cmd RegisterUserCommand) (string, error) {
	login := strings.ToLower(strings.TrimSpace(cmd.Login))
	if login == "" {
		login = "anonymous"
	}
	seed := fmt.Sprintf("%d-%s", time.Now().UnixNano(), login)
	inn, ogrn := buildValidRequisites(seed)
	companyName := "Индивидуальный учёт (" + login + ")"
	company, err := identity.NewCompany(uc.ids.NewCompanyID(), companyName, inn, ogrn, uc.clock.Now())
	if err != nil {
		return "", err
	}
	if err := uc.companies.Save(ctx, company); err != nil {
		return "", err
	}
	return company.ID(), nil
}

func buildValidRequisites(seed string) (string, string) {
	var digits []int
	for _, r := range seed {
		digits = append(digits, int(r)%10)
	}
	if len(digits) == 0 {
		digits = []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	}
	nextDigit := func(i int) int {
		return digits[i%len(digits)]
	}
	innBase := make([]int, 9)
	for i := range innBase {
		innBase[i] = nextDigit(i)
	}
	innChecksumWeights := []int{2, 4, 10, 3, 5, 9, 4, 6, 8}
	sum := 0
	for i, w := range innChecksumWeights {
		sum += innBase[i] * w
	}
	innChecksum := (sum % 11) % 10
	inn := ""
	for _, d := range append(innBase, innChecksum) {
		inn += strconv.Itoa(d)
	}

	ogrnBase := make([]int, 12)
	ogrnBase[0] = 1
	for i := 1; i < 12; i++ {
		ogrnBase[i] = nextDigit(i + 7)
	}
	baseStr := ""
	for _, d := range ogrnBase {
		baseStr += strconv.Itoa(d)
	}
	baseNum, _ := strconv.ParseInt(baseStr, 10, 64)
	ogrnChecksum := int((baseNum % 11) % 10)
	ogrn := baseStr + strconv.Itoa(ogrnChecksum)
	return inn, ogrn
}
