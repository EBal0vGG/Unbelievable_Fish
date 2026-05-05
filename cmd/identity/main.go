package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
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

	companyRepo := identitypg.NewCompanyRepository(db)
	userRepo := identitypg.NewUserRepository(db)
	passwordHasher := identityauth.NewPasswordHasher(0)
	tokenProvider := identityauth.NewTokenProvider(
		dbconfig.EnvOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		dbconfig.EnvDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)

	registerCompanyUC, err := identityapp.NewRegisterCompany(companyRepo, identityapp.NewRandomIDGenerator(), nil)
	if err != nil {
		logging.Fatal(logger, "register_company_usecase_init_failed", "error", err)
	}
	registerUserUC, err := identityapp.NewRegisterUser(userRepo, companyRepo, passwordHasher, identityapp.NewRandomIDGenerator(), nil)
	if err != nil {
		logging.Fatal(logger, "register_user_usecase_init_failed", "error", err)
	}
	loginUC, err := identityapp.NewLogin(userRepo, passwordHasher, tokenProvider)
	if err != nil {
		logging.Fatal(logger, "login_usecase_init_failed", "error", err)
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
	authMiddleware := handler.NewAuthMiddleware(tokenProvider)

	router := httpapi.NewRouter(httpapi.Handlers{
		RegisterCompany:  handler.NewRegisterCompanyHandler(registerCompanyUC),
		RegisterUser:     handler.NewRegisterUserHandler(registerUserUC),
		ListUsers:        authMiddleware.RequireRole(identity.RoleAdmin, handler.NewListUsersHandler(listUsersUC)),
		PromoteUserAdmin: authMiddleware.RequireRole(identity.RoleAdmin, handler.NewPromoteUserAdminHandler(promoteUserAdminUC)),
		Login:            handler.NewLoginHandler(loginUC),
		GetCurrentUser:   authMiddleware.Wrap(handler.NewGetCurrentUserHandler(getCurrentUserUC)),
	}, httplog.Middleware(logger))

	if err := ensureBootstrapAdmin(context.Background(), companyRepo, userRepo, passwordHasher); err != nil {
		logging.Fatal(logger, "bootstrap_admin_failed", "component", "bootstrap", "error", err)
	}

	port := dbconfig.EnvOrDefault("IDENTITY_PORT", "8084")
	logger.Info("http_server_starting", "component", "http.server", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logging.Fatal(logger, "http_server_stopped", "component", "http.server", "error", err)
	}
}

func ensureBootstrapAdmin(
	ctx context.Context,
	companyRepo *identitypg.CompanyRepository,
	userRepo *identitypg.UserRepository,
	hasher identityauth.PasswordHasher,
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
		if err := companyRepo.Save(ctx, company); err != nil {
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
	if err := user.AcceptTerms(termsVersion, time.Now().UTC()); err != nil {
		return err
	}
	if err := userRepo.Save(ctx, user); err != nil {
		return err
	}
	slog.Info("bootstrap_admin_created", "component", "bootstrap", "login", login, "company_id", company.ID())
	return nil
}
