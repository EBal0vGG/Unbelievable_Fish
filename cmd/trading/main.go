package main

import (
	"net/http"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httplog"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httpauth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http/handler"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
)

func main() {
	logger := logging.New("trading")
	db, ok := dbconfig.OpenPostgresFromEnv(0)
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()

	uow := tradingpg.NewUnitOfWork(db)
	publishAuctionUC, err := tradingapp.NewPublishAuction(uow)
	if err != nil {
		logging.Fatal(logger, "publish_auction_usecase_init_failed", "error", err)
	}
	placeBidUC, err := tradingapp.NewPlaceBid(uow)
	if err != nil {
		logging.Fatal(logger, "place_bid_usecase_init_failed", "error", err)
	}
	closeAuctionUC, err := tradingapp.NewCloseAuction(uow)
	if err != nil {
		logging.Fatal(logger, "close_auction_usecase_init_failed", "error", err)
	}
	cancelAuctionUC, err := tradingapp.NewCancelAuction(uow)
	if err != nil {
		logging.Fatal(logger, "cancel_auction_usecase_init_failed", "error", err)
	}
	getAuctionByLotUC, err := tradingapp.NewGetAuctionByLot(tradingpg.NewAuctionReadRepository(db))
	if err != nil {
		logging.Fatal(logger, "get_auction_by_lot_usecase_init_failed", "error", err)
	}
	getAuctionByIDUC, err := tradingapp.NewGetAuctionByID(tradingpg.NewAuctionReadRepository(db))
	if err != nil {
		logging.Fatal(logger, "get_auction_by_id_usecase_init_failed", "error", err)
	}
	tokenProvider := identityauth.NewTokenProvider(
		dbconfig.EnvOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		dbconfig.EnvDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)
	authMiddleware := identityauth.NewMiddleware(tokenProvider, httpauth.JSONErrorHandler("trading_auth_error"))

	router := httpapi.NewRouter(
		authMiddleware.RequireRole(identity.RoleSeller, handler.NewPublishAuctionHandler(publishAuctionUC)),
		authMiddleware.RequireRole(identity.RoleBuyer, handler.NewPlaceBidHandler(placeBidUC)),
		authMiddleware.RequireRole(identity.RoleSeller, handler.NewCloseAuctionHandler(closeAuctionUC)),
		authMiddleware.RequireRole(identity.RoleSeller, handler.NewCancelAuctionHandler(cancelAuctionUC)),
		authMiddleware.Wrap(handler.NewGetAuctionByIDHandler(getAuctionByIDUC)),
		authMiddleware.Wrap(handler.NewGetAuctionByLotHandler(getAuctionByLotUC)),
		httplog.Middleware(logger),
	)

	port := dbconfig.EnvOrDefault("TRADING_PORT", "8082")
	logger.Info("http_server_starting", "component", "http.server", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logging.Fatal(logger, "http_server_stopped", "component", "http.server", "error", err)
	}
}
