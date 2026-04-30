package main

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"
	"time"

	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/http/handler"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httplog"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httpauth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := logging.New("catalog")
	db, ok := openDB()
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()

	service := catalogapp.NewCatalogService(
		catalogpg.NewFishRepository(db),
		catalogpg.NewUnitRepository(db),
		catalogpg.NewProcessingTypeRepository(db),
		catalogpg.NewProductRepository(db),
		catalogpg.NewLotRepository(db),
		catalogpg.NewOutboxRepository(db),
		catalogapp.NewRandomIDGenerator(),
		catalogpg.NewTransactionManager(db, nil),
	)
	tokenProvider := identityauth.NewTokenProvider(
		envOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		time.Duration(envDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60))*time.Minute,
	)
	authMiddleware := identityauth.NewMiddleware(tokenProvider, httpauth.JSONErrorHandler("catalog_auth_error"))

	router := httpapi.NewRouter(
		handler.NewListFishHandler(service),
		handler.NewCreateFishHandler(service),
		handler.NewCreateProductHandler(service),
		handler.NewPublishProductHandler(service),
		authMiddleware.RequireRole(identity.RoleSeller, handler.NewCreateLotHandler(service)),
		authMiddleware.RequireRole(identity.RoleSeller, handler.NewPublishLotHandler(service)),
		httplog.Middleware(logger),
	)

	port := envOrDefault("CATALOG_PORT", "8081")
	logger.Info("http_server_starting", "component", "http.server", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logging.Fatal(logger, "http_server_stopped", "component", "http.server", "error", err)
	}
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

func envDurationMinutes(key string, def int) int {
	if value := os.Getenv(key); value != "" {
		if minutes, err := strconv.Atoi(value); err == nil && minutes > 0 {
			return minutes
		}
	}
	return def
}
