package main

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http/handler"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httplog"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := logging.New("deals")
	db, ok := openDB()
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()

	uow := dealspg.NewUnitOfWork(db)
	dealRepo := dealspg.NewDealRepository(db)
	projectionRepo := dealspg.NewProjectionRepository(db)
	confirmUC, err := dealsapp.NewConfirmDeal(uow)
	if err != nil {
		logging.Fatal(logger, "confirm_deal_usecase_init_failed", "error", err)
	}
	prepareUC, err := dealsapp.NewPrepareContract(uow)
	if err != nil {
		logging.Fatal(logger, "prepare_contract_usecase_init_failed", "error", err)
	}
	signUC, err := dealsapp.NewSignContract(uow)
	if err != nil {
		logging.Fatal(logger, "sign_contract_usecase_init_failed", "error", err)
	}
	requestPaymentUC, err := dealsapp.NewRequestPayment(uow)
	if err != nil {
		logging.Fatal(logger, "request_payment_usecase_init_failed", "error", err)
	}
	markPaidUC, err := dealsapp.NewMarkDealPaid(uow)
	if err != nil {
		logging.Fatal(logger, "mark_deal_paid_usecase_init_failed", "error", err)
	}
	requestShipmentUC, err := dealsapp.NewRequestShipment(uow)
	if err != nil {
		logging.Fatal(logger, "request_shipment_usecase_init_failed", "error", err)
	}
	markShippedUC, err := dealsapp.NewMarkDealShipped(uow)
	if err != nil {
		logging.Fatal(logger, "mark_deal_shipped_usecase_init_failed", "error", err)
	}
	completeUC, err := dealsapp.NewCompleteDeal(uow)
	if err != nil {
		logging.Fatal(logger, "complete_deal_usecase_init_failed", "error", err)
	}
	cancelUC, err := dealsapp.NewCancelDeal(uow)
	if err != nil {
		logging.Fatal(logger, "cancel_deal_usecase_init_failed", "error", err)
	}
	updatePriceUC, err := dealsapp.NewUpdateDealPrice(uow)
	if err != nil {
		logging.Fatal(logger, "update_deal_price_usecase_init_failed", "error", err)
	}
	tokenProvider := identityauth.NewTokenProvider(
		envOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		envDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)

	authMiddleware := identityauth.NewMiddleware(tokenProvider, func(w http.ResponseWriter, r *http.Request, err error) {
		httpErr := httpapi.MapError(err)
		slog.WarnContext(
			r.Context(),
			"deals_auth_error",
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
		handler.NewGetProjectionByAuctionIDHandler(dealsapp.NewGetProjectionByAuctionID(projectionRepo)),
		handler.NewGetDealByIDHandler(dealsapp.NewGetDealByID(dealRepo)),
		handler.NewGetDealByAuctionIDHandler(dealsapp.NewGetDealByAuctionID(dealRepo)),
		handler.NewConfirmDealHandler(confirmUC),
		handler.NewPrepareContractHandler(prepareUC),
		handler.NewSignContractHandler(signUC),
		handler.NewRequestPaymentHandler(requestPaymentUC),
		handler.NewMarkDealPaidHandler(markPaidUC),
		handler.NewRequestShipmentHandler(requestShipmentUC),
		handler.NewMarkDealShippedHandler(markShippedUC),
		handler.NewCompleteDealHandler(completeUC),
		handler.NewCancelDealHandler(cancelUC),
		handler.NewUpdateDealPriceHandler(updatePriceUC),
		httplog.Middleware(logger),
		authMiddleware.Wrap,
	)

	port := envOrDefault("DEALS_PORT", "8083")
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
