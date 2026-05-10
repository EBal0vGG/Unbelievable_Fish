package main

import (
	"context"
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

	fishRepo := catalogpg.NewFishRepository(db)

	service := catalogapp.NewCatalogService(
		fishRepo,
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

	if err := ensureBootstrapFish(context.Background(), fishRepo); err != nil {
		logging.Fatal(logger, "bootstrap_fish_failed", "component", "bootstrap", "error", err)
	}

	router := httpapi.NewRouter(httpapi.Handlers{
		ListFish:       handler.NewListFishHandler(service),
		CreateFish:     authMiddleware.RequireRole(identity.RoleAdmin, handler.NewCreateFishHandler(service)),
		CreateProduct:  authMiddleware.RequireRole(identity.RoleSeller, handler.NewCreateProductHandler(service)),
		PublishProduct: authMiddleware.RequireRole(identity.RoleSeller, handler.NewPublishProductHandler(service)),
		ListProducts:   authMiddleware.Wrap(handler.NewListProductsHandler(service)),
		CreateLot:      authMiddleware.RequireRole(identity.RoleSeller, handler.NewCreateLotHandler(service)),
		PublishLot:     authMiddleware.RequireRole(identity.RoleSeller, handler.NewPublishLotHandler(service)),
		ListLots:       authMiddleware.Wrap(handler.NewListLotsHandler(service)),
	}, httplog.Middleware(logger))

	port := dbconfig.EnvOrDefault("CATALOG_PORT", "8081")
	logger.Info("http_server_starting", "component", "http.server", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logging.Fatal(logger, "http_server_stopped", "component", "http.server", "error", err)
	}
}
