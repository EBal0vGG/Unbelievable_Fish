package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http/handler"
	identitypg "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, ok := openDB()
	if !ok {
		log.Fatal("PGHOST/PGUSER/PGDATABASE are required")
	}
	defer db.Close()

	companyRepo := identitypg.NewCompanyRepository(db)
	userRepo := identitypg.NewUserRepository(db)
	passwordHasher := identityauth.NewPasswordHasher(0)
	tokenProvider := identityauth.NewTokenProvider(
		envOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		envDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)

	registerCompanyUC, err := identityapp.NewRegisterCompany(companyRepo, identityapp.NewRandomIDGenerator(), nil)
	if err != nil {
		log.Fatal(err)
	}
	registerUserUC, err := identityapp.NewRegisterUser(userRepo, companyRepo, passwordHasher, identityapp.NewRandomIDGenerator(), nil)
	if err != nil {
		log.Fatal(err)
	}
	loginUC, err := identityapp.NewLogin(userRepo, passwordHasher, tokenProvider)
	if err != nil {
		log.Fatal(err)
	}
	getCurrentUserUC, err := identityapp.NewGetCurrentUser(userRepo)
	if err != nil {
		log.Fatal(err)
	}
	listUsersUC, err := identityapp.NewListUsers(userRepo)
	if err != nil {
		log.Fatal(err)
	}
	promoteUserAdminUC, err := identityapp.NewPromoteUserToAdmin(userRepo)
	if err != nil {
		log.Fatal(err)
	}
	authMiddleware := handler.NewAuthMiddleware(tokenProvider)

	router := httpapi.NewRouter(
		handler.NewRegisterCompanyHandler(registerCompanyUC),
		handler.NewRegisterUserHandler(registerUserUC),
		authMiddleware.RequireRole(identity.RoleAdmin, handler.NewListUsersHandler(listUsersUC)),
		authMiddleware.RequireRole(identity.RoleAdmin, handler.NewPromoteUserAdminHandler(promoteUserAdminUC)),
		handler.NewLoginHandler(loginUC),
		authMiddleware.Wrap(handler.NewGetCurrentUserHandler(getCurrentUserUC)),
	)

	if err := ensureBootstrapAdmin(context.Background(), companyRepo, userRepo, passwordHasher); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	port := envOrDefault("IDENTITY_PORT", "8084")
	log.Printf("identity http listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func openDB() (*sql.DB, bool) {
	host := os.Getenv("PGHOST")
	user := os.Getenv("PGUSER")
	password := os.Getenv("PGPASSWORD")
	database := os.Getenv("PGDATABASE")
	port := os.Getenv("PGPORT")
	sslmode := os.Getenv("PGSSLMODE")

	if host == "" || user == "" || database == "" {
		return nil, false
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := "host=" + host + " user=" + user + " dbname=" + database + " port=" + port + " sslmode=" + sslmode
	if password != "" {
		dsn += " password=" + password
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, false
	}
	db.SetMaxOpenConns(5)
	return db, true
}

func envOrDefault(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}

func envDurationMinutes(key string, def int) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return time.Duration(def) * time.Minute
	}
	minutes, err := strconv.Atoi(value)
	if err != nil || minutes <= 0 {
		return time.Duration(def) * time.Minute
	}
	return time.Duration(minutes) * time.Minute
}

func ensureBootstrapAdmin(
	ctx context.Context,
	companyRepo *identitypg.CompanyRepository,
	userRepo *identitypg.UserRepository,
	hasher identityauth.PasswordHasher,
) error {
	enabled := envOrDefault("IDENTITY_BOOTSTRAP_ADMIN_ENABLED", "true") != "false"
	if !enabled {
		return nil
	}

	login := envOrDefault("IDENTITY_BOOTSTRAP_ADMIN_LOGIN", "admin@fish.local")
	password := envOrDefault("IDENTITY_BOOTSTRAP_ADMIN_PASSWORD", "admin12345")
	companyID := envOrDefault("IDENTITY_BOOTSTRAP_ADMIN_COMPANY_ID", "company-admin")
	companyName := envOrDefault("IDENTITY_BOOTSTRAP_ADMIN_COMPANY_NAME", "Fish Platform Admin")
	userID := envOrDefault("IDENTITY_BOOTSTRAP_ADMIN_USER_ID", "user-admin")
	userName := envOrDefault("IDENTITY_BOOTSTRAP_ADMIN_USER_NAME", "Platform Administrator")
	inn := envOrDefault("IDENTITY_BOOTSTRAP_ADMIN_INN", "7707083893")
	ogrn := envOrDefault("IDENTITY_BOOTSTRAP_ADMIN_OGRN", "1027700132195")
	termsVersion := envOrDefault("IDENTITY_BOOTSTRAP_ADMIN_TERMS_VERSION", "v1")

	exists, err := userRepo.ExistsByLogin(ctx, login)
	if err != nil {
		return err
	}
	if exists {
		log.Printf("bootstrap_admin_exists login=%s", login)
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
	log.Printf("bootstrap_admin_created login=%s company_id=%s", login, company.ID())
	return nil
}
