package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http/handler"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, ok := openDB()
	if !ok {
		log.Fatal("PGHOST/PGUSER/PGDATABASE are required")
	}
	defer db.Close()

	uow := tradingpg.NewUnitOfWork(db)
	createAuctionUC, err := tradingapp.NewCreateAuction(uow, tradingapp.RandomAuctionIDFactory{})
	if err != nil {
		log.Fatal(err)
	}
	publishAuctionUC, err := tradingapp.NewPublishAuction(uow)
	if err != nil {
		log.Fatal(err)
	}
	placeBidUC, err := tradingapp.NewPlaceBid(uow)
	if err != nil {
		log.Fatal(err)
	}
	closeAuctionUC, err := tradingapp.NewCloseAuction(uow)
	if err != nil {
		log.Fatal(err)
	}
	cancelAuctionUC, err := tradingapp.NewCancelAuction(uow)
	if err != nil {
		log.Fatal(err)
	}
	getAuctionByLotUC, err := tradingapp.NewGetAuctionByLot(tradingpg.NewAuctionReadRepository(db))
	if err != nil {
		log.Fatal(err)
	}
	getAuctionByIDUC, err := tradingapp.NewGetAuctionByID(tradingpg.NewAuctionReadRepository(db))
	if err != nil {
		log.Fatal(err)
	}
	tokenProvider := identityauth.NewTokenProvider(
		envOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		envDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)
	authMiddleware := identityauth.NewMiddleware(tokenProvider, func(w http.ResponseWriter, r *http.Request, err error) {
		httpErr := httpapi.MapError(err)
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
	)

	port := envOrDefault("TRADING_PORT", "8082")
	log.Printf("trading http listening on :%s", port)
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
