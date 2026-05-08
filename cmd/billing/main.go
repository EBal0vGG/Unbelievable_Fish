package main

import (
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/http/handler"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/payment/fake"
	billingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/postgres"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httpauth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httplog"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
)

func main() {
	logger := logging.New("billing")
	db, ok := dbconfig.OpenPostgresFromEnv(0)
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()

	returnURL := dbconfig.EnvOrDefault("BILLING_TOP_UP_RETURN_URL", "http://localhost")

	txm := billingpg.NewTransactionManager(db, nil)
	accounts := billingpg.NewAccountRepository(db)
	ledger := billingpg.NewLedgerRepository(db)
	processed := billingpg.NewProcessedTopUpRepository(db)
	topUps := billingpg.NewTopUpRepository(db)
	ledgerLister := billingpg.NewLedgerLister(db)
	deposits := billingpg.NewAuctionDepositRepository(db)
	events := billingpg.NewOutboxRepository(db)

	createAccount, err := billingapp.NewCreateAccount(accounts, billingapp.RandomHexID{}, events)
	if err != nil {
		logging.Fatal(logger, "create_account_init_failed", "error", err)
	}
	confirmTopUp, err := billingapp.NewConfirmTopUp(accounts, ledger, processed, billingapp.RandomHexID{}, nil, events)
	if err != nil {
		logging.Fatal(logger, "confirm_top_up_init_failed", "error", err)
	}

	fakePayment := fake.Provider{}
	createTopUpUC, err := billingapp.NewCreateTopUp(
		createAccount,
		accounts,
		topUps,
		fakePayment,
		fake.ProviderName,
		billingapp.RandomHexID{},
		nil,
		returnURL,
	)
	if err != nil {
		logging.Fatal(logger, "create_top_up_init_failed", "error", err)
	}
	confirmTopUpByProvider, err := billingapp.NewConfirmTopUpByProvider(topUps, confirmTopUp, nil)
	if err != nil {
		logging.Fatal(logger, "confirm_top_up_by_provider_init_failed", "error", err)
	}

	tokenProvider := identityauth.NewTokenProvider(
		dbconfig.EnvOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		dbconfig.EnvDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)
	authMiddleware := identityauth.NewMiddleware(tokenProvider, httpauth.JSONErrorHandler("billing_auth_error"))

	inner := httpapi.NewRouter(httpapi.Handlers{
		GetBalance:       authMiddleware.Wrap(handler.NewGetBalanceHandler(txm, accounts, createAccount)),
		TestTopUp:        authMiddleware.Wrap(handler.NewTestTopUpHandler(txm, createAccount, confirmTopUp, billingapp.RandomHexID{})),
		GetLedger:        authMiddleware.Wrap(handler.NewGetLedgerHandler(ledgerLister)),
		GetDeposits:      authMiddleware.Wrap(handler.NewGetDepositsHandler(deposits)),
		CreateTopUp:      authMiddleware.Wrap(handler.NewCreateTopUpHandler(txm, createTopUpUC)),
		ListTopUps:       authMiddleware.Wrap(handler.NewListTopUpsHandler(topUps)),
		FakeConfirmTopUp: authMiddleware.Wrap(handler.NewFakeConfirmTopUpHandler(txm, confirmTopUpByProvider)),
	}, httplog.Middleware(logger))

	r := http.NewServeMux()
	r.Handle("/billing/", http.StripPrefix("/billing", inner))

	port := dbconfig.EnvOrDefault("BILLING_PORT", "8085")
	logger.Info("http_server_starting", "component", "http.server", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logging.Fatal(logger, "http_server_stopped", "component", "http.server", "error", err)
	}
}
