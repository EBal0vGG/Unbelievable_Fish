package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http/handler"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, ok := openDB()
	if !ok {
		log.Fatal("PGHOST/PGUSER/PGDATABASE are required")
	}
	defer db.Close()

	uow := dealspg.NewUnitOfWork(db)
	dealRepo := dealspg.NewDealRepository(db)
	projectionRepo := dealspg.NewProjectionRepository(db)
	confirmUC, err := dealsapp.NewConfirmDeal(uow)
	if err != nil {
		log.Fatal(err)
	}
	prepareUC, err := dealsapp.NewPrepareContract(uow)
	if err != nil {
		log.Fatal(err)
	}
	signUC, err := dealsapp.NewSignContract(uow)
	if err != nil {
		log.Fatal(err)
	}
	requestPaymentUC, err := dealsapp.NewRequestPayment(uow)
	if err != nil {
		log.Fatal(err)
	}
	markPaidUC, err := dealsapp.NewMarkDealPaid(uow)
	if err != nil {
		log.Fatal(err)
	}
	requestShipmentUC, err := dealsapp.NewRequestShipment(uow)
	if err != nil {
		log.Fatal(err)
	}
	markShippedUC, err := dealsapp.NewMarkDealShipped(uow)
	if err != nil {
		log.Fatal(err)
	}
	completeUC, err := dealsapp.NewCompleteDeal(uow)
	if err != nil {
		log.Fatal(err)
	}
	cancelUC, err := dealsapp.NewCancelDeal(uow)
	if err != nil {
		log.Fatal(err)
	}
	updatePriceUC, err := dealsapp.NewUpdateDealPrice(uow)
	if err != nil {
		log.Fatal(err)
	}

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
	)

	port := envOrDefault("DEALS_PORT", "8083")
	log.Printf("deals http listening on :%s", port)
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
