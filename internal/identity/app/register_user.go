package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type RegisterUser struct {
	users             UserRepository
	companies         CompanyRepository
	hasher            PasswordHasher
	ids               IDGenerator
	clock             Clock
	uow               UnitOfWork
	publisher         CompanyCreatedPublisher
	emailVerification *EmailVerificationService
	autoVerifyEmail   bool
}

func NewRegisterUser(
	users UserRepository,
	companies CompanyRepository,
	hasher PasswordHasher,
	ids IDGenerator,
	clock Clock,
	uow UnitOfWork,
	publisher CompanyCreatedPublisher,
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
		uow:       uow,
		publisher: publisher,
	}, nil
}

func (uc *RegisterUser) WithEmailVerification(service *EmailVerificationService) {
	uc.emailVerification = service
}

func (uc *RegisterUser) WithAutoVerifyEmail() {
	uc.autoVerifyEmail = true
}

func (uc *RegisterUser) Execute(ctx context.Context, cmd RegisterUserCommand) (UserDTO, error) {
	slog.InfoContext(ctx, "register_request_started", "component", "identity.register_user")
	if strings.TrimSpace(cmd.Password) == "" {
		slog.WarnContext(ctx, "register_request_failed", "component", "identity.register_user", "code", "PASSWORD_REQUIRED")
		return UserDTO{}, ErrPasswordRequired
	}
	if !cmd.AcceptedTerms {
		slog.WarnContext(ctx, "register_request_failed", "component", "identity.register_user", "code", "TERMS_ACCEPTANCE_REQUIRED")
		return UserDTO{}, ErrTermsAcceptanceRequired
	}
	if strings.TrimSpace(cmd.TermsVersion) == "" {
		slog.WarnContext(ctx, "register_request_failed", "component", "identity.register_user", "code", "TERMS_VERSION_REQUIRED")
		return UserDTO{}, ErrTermsVersionRequired
	}
	if cmd.Role == identity.RoleAdmin {
		slog.WarnContext(ctx, "register_request_failed", "component", "identity.register_user", "code", "ADMIN_REGISTRATION_FORBIDDEN")
		return UserDTO{}, ErrAdminRegistrationForbidden
	}

	login := strings.ToLower(strings.TrimSpace(cmd.Login))
	existing, err := uc.users.GetByLogin(ctx, login)
	if err == nil && existing != nil {
		slog.WarnContext(ctx, "register_request_failed", "component", "identity.register_user", "code", "LOGIN_ALREADY_USED")
		return UserDTO{}, ErrLoginAlreadyUsed
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		slog.ErrorContext(ctx, "register_request_failed", "component", "identity.register_user", "error", err)
		return UserDTO{}, err
	}

	companyID, newCompany, err := uc.resolveRegistrationCompany(ctx, cmd)
	if err != nil {
		slog.ErrorContext(ctx, "register_request_failed", "component", "identity.register_user", "error", err)
		return UserDTO{}, err
	}

	passwordHash, err := uc.hasher.Hash(cmd.Password)
	if err != nil {
		slog.ErrorContext(ctx, "register_request_failed", "component", "identity.register_user", "error", err)
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
		slog.WarnContext(ctx, "register_request_failed", "component", "identity.register_user", "error", err)
		return UserDTO{}, err
	}
	if err := user.AcceptTerms(cmd.TermsVersion, uc.clock.Now()); err != nil {
		slog.WarnContext(ctx, "register_request_failed", "component", "identity.register_user", "error", err)
		return UserDTO{}, err
	}
	if uc.autoVerifyEmail {
		user.VerifyEmail()
	}

	var verificationEmail VerificationEmail
	save := func(saveCtx context.Context) error {
		if newCompany != nil {
			slog.InfoContext(saveCtx, "company_create_started", "component", "identity.register_user", "company_id", newCompany.ID())
			if err := uc.companies.Save(saveCtx, newCompany); err != nil {
				return err
			}
			if uc.publisher != nil {
				if err := uc.publisher.PublishCompanyCreated(saveCtx, newCompany.ID()); err != nil {
					return err
				}
			}
			slog.InfoContext(saveCtx, "company_create_succeeded", "component", "identity.register_user", "company_id", newCompany.ID())
		}

		slog.InfoContext(saveCtx, "user_create_started", "component", "identity.register_user", "user_id", user.ID())
		if err := uc.users.Save(saveCtx, user); err != nil {
			return err
		}
		slog.InfoContext(saveCtx, "user_create_succeeded", "component", "identity.register_user", "user_id", user.ID())

		if uc.emailVerification != nil {
			slog.InfoContext(saveCtx, "verification_token_create_started", "component", "identity.register_user", "user_id", user.ID())
			email, err := uc.emailVerification.CreateToken(saveCtx, user)
			if err != nil {
				return err
			}
			verificationEmail = email
			slog.InfoContext(saveCtx, "verification_token_create_succeeded", "component", "identity.register_user", "user_id", user.ID())
		}
		return nil
	}
	if uc.uow != nil {
		if err := uc.uow.WithinTx(ctx, save); err != nil {
			slog.ErrorContext(ctx, "register_request_failed", "component", "identity.register_user", "error", err)
			return UserDTO{}, err
		}
	} else if err := save(ctx); err != nil {
		slog.ErrorContext(ctx, "register_request_failed", "component", "identity.register_user", "error", err)
		return UserDTO{}, err
	}

	if uc.emailVerification != nil {
		slog.InfoContext(ctx, "email_send_started", "component", "identity.register_user", "user_id", user.ID())
		if err := uc.emailVerification.SendEmail(ctx, user.ID(), verificationEmail); err != nil {
			slog.ErrorContext(ctx, "email_send_failed", "component", "identity.register_user", "user_id", user.ID(), "error", err)
			slog.WarnContext(ctx, "register_request_failed", "component", "identity.register_user", "code", "EMAIL_SEND_FAILED", "user_id", user.ID())
			return UserDTO{}, err
		}
		slog.InfoContext(ctx, "email_send_succeeded", "component", "identity.register_user", "user_id", user.ID())
	}
	slog.InfoContext(ctx, "register_request_completed", "component", "identity.register_user", "user_id", user.ID())
	return userDTOFromDomain(user), nil
}

func (uc *RegisterUser) resolveRegistrationCompany(ctx context.Context, cmd RegisterUserCommand) (string, *identity.Company, error) {
	companyID := strings.TrimSpace(cmd.CompanyID)
	if companyID != "" {
		if _, err := uc.companies.GetByID(ctx, companyID); err != nil {
			if errors.Is(err, ErrCompanyNotFound) {
				return "", nil, ErrCompanyNotFound
			}
			return "", nil, err
		}
		return companyID, nil, nil
	}

	companyINN := strings.TrimSpace(cmd.CompanyINN)
	companyOGRN := strings.TrimSpace(cmd.CompanyOGRN)
	if companyINN != "" && companyOGRN != "" {
		company, err := uc.companies.GetByRequisites(ctx, companyINN, companyOGRN)
		if err != nil {
			if errors.Is(err, ErrCompanyNotFound) {
				return "", nil, ErrCompanyNotFound
			}
			return "", nil, err
		}
		return company.ID(), nil, nil
	}

	company, err := uc.buildDummyCompany(cmd)
	if err != nil {
		return "", nil, err
	}
	return company.ID(), company, nil
}

func (uc *RegisterUser) buildDummyCompany(cmd RegisterUserCommand) (*identity.Company, error) {
	login := strings.ToLower(strings.TrimSpace(cmd.Login))
	if login == "" {
		login = "anonymous"
	}
	seed := fmt.Sprintf("%d-%s", time.Now().UnixNano(), login)
	inn, ogrn := buildValidRequisites(seed)
	companyName := "Индивидуальный учёт (" + login + ")"
	return identity.NewCompany(uc.ids.NewCompanyID(), companyName, inn, ogrn, uc.clock.Now())
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
