package main

import (
	"net/http"

	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/http/handler"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httplog"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httpauth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
)

func main() {
	logger := logging.New("catalog")
	db, ok := dbconfig.OpenPostgresFromEnv(0)
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
		dbconfig.EnvOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		dbconfig.EnvDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
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

	port := dbconfig.EnvOrDefault("CATALOG_PORT", "8081")
	logger.Info("http_server_starting", "component", "http.server", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logging.Fatal(logger, "http_server_stopped", "component", "http.server", "error", err)
	}
}
