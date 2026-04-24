package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
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

	router := httpapi.NewRouter(
		handler.NewRegisterCompanyHandler(registerCompanyUC),
		handler.NewRegisterUserHandler(registerUserUC),
		handler.NewLoginHandler(loginUC),
		handler.NewAuthMiddleware(tokenProvider).Wrap(handler.NewGetCurrentUserHandler(getCurrentUserUC)),
	)

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
