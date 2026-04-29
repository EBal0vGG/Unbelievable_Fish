package main

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httplog"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http/handler"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := logging.New("trading")
	db, ok := openDB()
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()

	uow := tradingpg.NewUnitOfWork(db)
	createAuctionUC, err := tradingapp.NewCreateAuction(uow, tradingapp.RandomAuctionIDFactory{})
	if err != nil {
		logging.Fatal(logger, "create_auction_usecase_init_failed", "error", err)
	}
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
		envOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		envDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)
	authMiddleware := identityauth.NewMiddleware(tokenProvider, func(w http.ResponseWriter, r *http.Request, err error) {
		httpErr := httpapi.MapError(err)
		slog.WarnContext(
			r.Context(),
			"trading_auth_error",
			"component", "auth.middleware",
			"status", httpErr.Status,
			"code", httpErr.Code,
			"message", httpErr.Message,
			"correlation_id", r.Header.Get("X-Correlation-ID"),
			"causation_id", r.Header.Get("X-Causation-ID"),
			"error", err,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpErr.Status)
		_ = json.NewEncoder(w).Encode(httpapi.ErrorResponse{
			Code:    httpErr.Code,
			Message: httpErr.Message,
		})
	})

	router := httpapi.NewRouter(
		authMiddleware.RequireRole(identity.RoleSeller, handler.NewCreateAuctionHandler(createAuctionUC)),
		authMiddleware.RequireRole(identity.RoleSeller, handler.NewPublishAuctionHandler(publishAuctionUC)),
		authMiddleware.RequireRole(identity.RoleBuyer, handler.NewPlaceBidHandler(placeBidUC)),
		authMiddleware.RequireRole(identity.RoleSeller, handler.NewCloseAuctionHandler(closeAuctionUC)),
		authMiddleware.RequireRole(identity.RoleSeller, handler.NewCancelAuctionHandler(cancelAuctionUC)),
		authMiddleware.Wrap(handler.NewGetAuctionByIDHandler(getAuctionByIDUC)),
		authMiddleware.Wrap(handler.NewGetAuctionByLotHandler(getAuctionByLotUC)),
		httplog.Middleware(logger),
	)

	port := envOrDefault("TRADING_PORT", "8082")
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
