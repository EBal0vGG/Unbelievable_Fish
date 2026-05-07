package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httpauth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httplog"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
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
	listAuctionsUC, err := tradingapp.NewListAuctions(tradingpg.NewAuctionReadRepository(db))
	if err != nil {
		logging.Fatal(logger, "list_auctions_usecase_init_failed", "error", err)
	}
	tokenProvider := identityauth.NewTokenProvider(
		dbconfig.EnvOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		dbconfig.EnvDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)
	authMiddleware := identityauth.NewMiddleware(tokenProvider, httpauth.JSONErrorHandler("trading_auth_error"))

	router := httpapi.NewRouter(httpapi.Handlers{
		ListAuctions:   authMiddleware.Wrap(handler.NewListAuctionsHandler(listAuctionsUC)),
		PublishAuction: authMiddleware.RequireRole(identity.RoleSeller, handler.NewPublishAuctionHandler(publishAuctionUC)),
		PlaceBid:       authMiddleware.RequireRole(identity.RoleBuyer, handler.NewPlaceBidHandler(placeBidUC)),
		CloseAuction:   authMiddleware.RequireRole(identity.RoleSeller, handler.NewCloseAuctionHandler(closeAuctionUC)),
		CancelAuction:  authMiddleware.RequireRole(identity.RoleSeller, handler.NewCancelAuctionHandler(cancelAuctionUC)),
		GetByID:        authMiddleware.Wrap(handler.NewGetAuctionByIDHandler(getAuctionByIDUC)),
		GetByLot:       authMiddleware.Wrap(handler.NewGetAuctionByLotHandler(getAuctionByLotUC)),
	}, httplog.Middleware(logger))

	if envBool("TRADING_CLOSE_EXPIRED_ENABLED", false) {
		startExpiredAuctionCloser(logger, closeAuctionUC, tradingpg.NewAuctionLister(db))
		logger.Info("trading_close_expired_scheduler_enabled", "component", "scheduler")
	} else {
		logger.Info("trading_close_expired_scheduler_disabled", "component", "scheduler")
	}

	port := dbconfig.EnvOrDefault("TRADING_PORT", "8082")
	logger.Info("http_server_starting", "component", "http.server", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logging.Fatal(logger, "http_server_stopped", "component", "http.server", "error", err)
	}
}

type expiredAuctionLister interface {
	ListExpired(ctx context.Context, now time.Time, limit int) ([]tradingapp.AuctionID, error)
}

func startExpiredAuctionCloser(logger *slog.Logger, closeAuctionUC *tradingapp.CloseAuction, lister expiredAuctionLister) {
	interval := envDurationSeconds("TRADING_CLOSE_EXPIRED_INTERVAL_SEC", 5)
	limit := envInt("TRADING_CLOSE_EXPIRED_BATCH", 100)
	ctx := context.Background()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if err := closeExpiredAuctionsOnce(ctx, closeAuctionUC, lister, limit); err != nil {
				logger.Error("trading_close_expired_failed", "component", "scheduler", "error", err)
			}
			<-ticker.C
		}
	}()
}

func closeExpiredAuctionsOnce(
	ctx context.Context,
	closeAuctionUC *tradingapp.CloseAuction,
	lister expiredAuctionLister,
	limit int,
) error {
	ids, err := lister.ListExpired(ctx, time.Now().UTC(), limit)
	if err != nil {
		return err
	}
	for _, id := range ids {
		meta := tradingapp.CommandMeta{
			CompanyID:     "system",
			UserID:        "system",
			CorrelationID: "trading-close-expired",
			CausationID:   "trading-close-expired",
		}
		if err := closeAuctionUC.Execute(ctx, meta, id); err != nil {
			if errors.Is(err, tradingapp.ErrNotFound) ||
				errors.Is(err, auction.ErrCannotCloseAuction) ||
				errors.Is(err, auction.ErrAuctionNotActive) ||
				errors.Is(err, auction.ErrInvalidStateTransition) {
				continue
			}
			return err
		}
	}
	return nil
}

func envDurationSeconds(key string, def int) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return time.Duration(def) * time.Second
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Duration(def) * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func envInt(key string, def int) int {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return def
	}
	return parsed
}

func envBool(key string, def bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return def
	}
	return parsed
}
