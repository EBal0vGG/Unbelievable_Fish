package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	identityemail "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/email"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http/handler"
	identitypg "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/postgres"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httplog"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
)

func main() {
	logger := logging.New("identity")
	db, ok := dbconfig.OpenPostgresFromEnv(0)
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()
	if err := waitForDatabase(context.Background(), db, logger, 60*time.Second); err != nil {
		logging.Fatal(logger, "db_connect_timeout", "component", "startup", "error", err)
	}

	companyRepo := identitypg.NewCompanyRepository(db)
	userRepo := identitypg.NewUserRepository(db)
	emailTokenRepo := identitypg.NewEmailVerificationTokenRepository(db)
	passwordHasher := identityauth.NewPasswordHasher(0)
	tokenProvider := identityauth.NewTokenProvider(
		dbconfig.EnvOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		dbconfig.EnvDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)

	txManager := identitypg.NewTransactionManager(db, nil)
	outboxRepo := identitypg.NewOutboxRepository(db)
	verificationTokenGenerator := identityapp.NewSecureVerificationTokenGenerator()
	verificationEmailSender := identityemail.NewSenderFromEnv(logger)
	emailVerificationService, err := identityapp.NewEmailVerificationService(
		emailTokenRepo,
		verificationEmailSender,
		verificationTokenGenerator,
		dbconfig.EnvOrDefault("APP_PUBLIC_URL", "http://localhost:3000"),
		dbconfig.EnvDurationMinutes("EMAIL_VERIFICATION_TTL_MINUTES", 24*60),
		dbconfig.EnvDurationMinutes("EMAIL_VERIFICATION_COOLDOWN_MINUTES", 5),
		nil,
	)
	if err != nil {
		logging.Fatal(logger, "email_verification_service_init_failed", "error", err)
	}

	registerCompanyUC, err := identityapp.NewRegisterCompany(companyRepo, identityapp.NewRandomIDGenerator(), nil, txManager, outboxRepo)
	if err != nil {
		logging.Fatal(logger, "register_company_usecase_init_failed", "error", err)
	}
	registerCompanyUC.WithCompanyVerifier(identityapp.NewNoopCompanyVerifier())
	registerUserUC, err := identityapp.NewRegisterUser(userRepo, companyRepo, passwordHasher, identityapp.NewRandomIDGenerator(), nil, txManager, outboxRepo)
	if err != nil {
		logging.Fatal(logger, "register_user_usecase_init_failed", "error", err)
	}
	emailVerificationEnabled := dbconfig.EnvBool("IDENTITY_EMAIL_VERIFICATION_ENABLED", true)
	if emailVerificationEnabled {
		registerUserUC.WithEmailVerification(emailVerificationService)
	} else {
		registerUserUC.WithAutoVerifyEmail()
		logger.Warn("email_verification_disabled", "component", "startup", "env", "IDENTITY_EMAIL_VERIFICATION_ENABLED")
	}
	loginUC, err := identityapp.NewLogin(userRepo, passwordHasher, tokenProvider)
	if err != nil {
		logging.Fatal(logger, "login_usecase_init_failed", "error", err)
	}
	if !emailVerificationEnabled {
		loginUC.WithSkipEmailVerification()
	}
	getCurrentUserUC, err := identityapp.NewGetCurrentUser(userRepo)
	if err != nil {
		logging.Fatal(logger, "get_current_user_usecase_init_failed", "error", err)
	}
	listUsersUC, err := identityapp.NewListUsers(userRepo)
	if err != nil {
		logging.Fatal(logger, "list_users_usecase_init_failed", "error", err)
	}
	promoteUserAdminUC, err := identityapp.NewPromoteUserToAdmin(userRepo)
	if err != nil {
		logging.Fatal(logger, "promote_user_admin_usecase_init_failed", "error", err)
	}
	var verifyEmailHandler http.Handler
	var resendVerificationHandler http.Handler
	if emailVerificationEnabled {
		verifyEmailUC, err := identityapp.NewVerifyEmail(userRepo, emailTokenRepo, verificationTokenGenerator, nil, txManager)
		if err != nil {
			logging.Fatal(logger, "verify_email_usecase_init_failed", "error", err)
		}
		resendVerificationUC, err := identityapp.NewResendVerification(userRepo, emailVerificationService)
		if err != nil {
			logging.Fatal(logger, "resend_verification_usecase_init_failed", "error", err)
		}
		verifyEmailHandler = handler.NewVerifyEmailHandler(verifyEmailUC)
		resendVerificationHandler = handler.NewResendVerificationHandler(resendVerificationUC)
	}
	authMiddleware := handler.NewAuthMiddleware(tokenProvider)

	router := httpapi.NewRouter(httpapi.Handlers{
		RegisterCompany:    handler.NewRegisterCompanyHandler(registerCompanyUC),
		RegisterUser:       handler.NewRegisterUserHandler(registerUserUC),
		ListUsers:          authMiddleware.RequireRole(identity.RoleAdmin, handler.NewListUsersHandler(listUsersUC)),
		PromoteUserAdmin:   authMiddleware.RequireRole(identity.RoleAdmin, handler.NewPromoteUserAdminHandler(promoteUserAdminUC)),
		Login:              handler.NewLoginHandler(loginUC),
		VerifyEmail:        verifyEmailHandler,
		ResendVerification: resendVerificationHandler,
		GetCurrentUser:     authMiddleware.Wrap(handler.NewGetCurrentUserHandler(getCurrentUserUC)),
	}, httplog.Middleware(logger))

	if err := ensureBootstrapAdmin(context.Background(), companyRepo, userRepo, passwordHasher, txManager, outboxRepo); err != nil {
		logging.Fatal(logger, "bootstrap_admin_failed", "component", "bootstrap", "error", err)
	}

	port := dbconfig.EnvOrDefault("IDENTITY_PORT", "8084")
	logger.Info("http_server_starting", "component", "http.server", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logging.Fatal(logger, "http_server_stopped", "component", "http.server", "error", err)
	}
}

func waitForDatabase(ctx context.Context, db *sql.DB, logger *slog.Logger, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	backoff := time.Second
	attempt := 1
	for {
		pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
		err := db.PingContext(pingCtx)
		pingCancel()
		if err == nil {
			logger.Info("db_connect_succeeded", "component", "startup", "attempt", attempt)
			return nil
		}
		logger.Warn("db_connect_attempt_failed", "component", "startup", "attempt", attempt, "error", err)

		select {
		case <-ctx.Done():
			logger.Error("db_connect_timeout", "component", "startup", "attempts", attempt, "error", ctx.Err())
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 2*time.Second {
			backoff *= 2
		}
		attempt++
	}
}

func ensureBootstrapAdmin(
	ctx context.Context,
	companyRepo *identitypg.CompanyRepository,
	userRepo *identitypg.UserRepository,
	hasher identityauth.PasswordHasher,
	txManager *identitypg.TransactionManager,
	outboxRepo *identitypg.OutboxRepository,
) error {
	enabled := dbconfig.EnvOrDefault("IDENTITY_BOOTSTRAP_ADMIN_ENABLED", "true") != "false"
	if !enabled {
		return nil
	}

	login := dbconfig.EnvOrDefault("IDENTITY_BOOTSTRAP_ADMIN_LOGIN", "admin@fish.local")
	password := dbconfig.EnvOrDefault("IDENTITY_BOOTSTRAP_ADMIN_PASSWORD", "admin12345")
	companyID := dbconfig.EnvOrDefault("IDENTITY_BOOTSTRAP_ADMIN_COMPANY_ID", "company-admin")
	companyName := dbconfig.EnvOrDefault("IDENTITY_BOOTSTRAP_ADMIN_COMPANY_NAME", "Fish Platform Admin")
	userID := dbconfig.EnvOrDefault("IDENTITY_BOOTSTRAP_ADMIN_USER_ID", "user-admin")
	userName := dbconfig.EnvOrDefault("IDENTITY_BOOTSTRAP_ADMIN_USER_NAME", "Platform Administrator")
	inn := dbconfig.EnvOrDefault("IDENTITY_BOOTSTRAP_ADMIN_INN", "7707083893")
	ogrn := dbconfig.EnvOrDefault("IDENTITY_BOOTSTRAP_ADMIN_OGRN", "1027700132195")
	termsVersion := dbconfig.EnvOrDefault("IDENTITY_BOOTSTRAP_ADMIN_TERMS_VERSION", "v1")

	exists, err := userRepo.ExistsByLogin(ctx, login)
	if err != nil {
		return err
	}
	if exists {
		slog.Info("bootstrap_admin_exists", "component", "bootstrap", "login", login)
		return nil
	}

	company, err := companyRepo.GetByRequisites(ctx, inn, ogrn)
	if err != nil {
		if !errors.Is(err, identityapp.ErrCompanyNotFound) {
			return err
		}
		company, err = identity.NewCompany(companyID, companyName, inn, ogrn, time.Now().UTC())
		if err != nil {
			return err
		}
		if err := txManager.WithinTx(ctx, func(txCtx context.Context) error {
			if err := companyRepo.Save(txCtx, company); err != nil {
				return err
			}
			return outboxRepo.PublishCompanyCreated(txCtx, company.ID())
		}); err != nil {
			return err
		}
	}

	passwordHash, err := hasher.Hash(password)
	if err != nil {
		return err
	}
	user, err := identity.NewUser(userID, company.ID(), userName, identity.RoleAdmin, login, passwordHash)
	if err != nil {
		return err
	}
	user.VerifyEmail()
	if err := user.AcceptTerms(termsVersion, time.Now().UTC()); err != nil {
		return err
	}
	if err := userRepo.Save(ctx, user); err != nil {
		return err
	}
	slog.Info("bootstrap_admin_created", "component", "bootstrap", "login", login, "company_id", company.ID())
	return nil
}
