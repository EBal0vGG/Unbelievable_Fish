package main

import (
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/http/handler"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/payment/fake"
	billingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/postgres"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
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
	dealInvoices := billingpg.NewDealInvoiceRepository(db)
	sellerPayouts := billingpg.NewSellerPayoutRepository(db)
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

	confirmDealInvoice, err := billingapp.NewConfirmDealInvoicePaid(dealInvoices, events, nil)
	if err != nil {
		logging.Fatal(logger, "confirm_deal_invoice_init_failed", "error", err)
	}

	expireDealInvoice, err := billingapp.NewExpireDealInvoice(dealInvoices, events, nil)
	if err != nil {
		logging.Fatal(logger, "expire_deal_invoice_init_failed", "error", err)
	}

	markPayoutReady, err := billingapp.NewMarkSellerPayoutReady(sellerPayouts, nil, events)
	if err != nil {
		logging.Fatal(logger, "mark_seller_payout_ready_init_failed", "error", err)
	}
	markPayoutPaid, err := billingapp.NewMarkSellerPayoutPaid(
		sellerPayouts,
		accounts,
		ledger,
		createAccount,
		billingapp.RandomHexID{},
		nil,
		events,
	)
	if err != nil {
		logging.Fatal(logger, "mark_seller_payout_paid_init_failed", "error", err)
	}

	tokenProvider := identityauth.NewTokenProvider(
		dbconfig.EnvOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		dbconfig.EnvDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)
	authMiddleware := identityauth.NewMiddleware(tokenProvider, httpauth.JSONErrorHandler("billing_auth_error"))

	var notFound http.Handler = http.NotFoundHandler()
	enableFake := dbconfig.EnvBool("BILLING_ENABLE_FAKE_PROVIDER", false)
	enableAdmin := dbconfig.EnvBool("BILLING_ENABLE_ADMIN_ACTIONS", false)

	testTopUpH := notFound
	fakeTopUpH := notFound
	fakeInvoiceH := notFound
	if enableFake {
		testTopUpH = authMiddleware.Wrap(handler.NewTestTopUpHandler(txm, createAccount, confirmTopUp, billingapp.RandomHexID{}))
		fakeTopUpH = authMiddleware.Wrap(handler.NewFakeConfirmTopUpHandler(txm, confirmTopUpByProvider))
		fakeInvoiceH = authMiddleware.Wrap(handler.NewFakeConfirmDealInvoiceHandler(txm, confirmDealInvoice))
	}

	adminConfirmInv := notFound
	adminExpireInv := notFound
	adminPayoutReady := notFound
	adminPayoutPaid := notFound
	if enableAdmin {
		adminConfirmInv = authMiddleware.RequireRole(identity.RoleAdmin, handler.NewAdminConfirmDealInvoiceHandler(txm, confirmDealInvoice))
		adminExpireInv = authMiddleware.RequireRole(identity.RoleAdmin, handler.NewAdminExpireDealInvoiceHandler(txm, expireDealInvoice))
		adminPayoutReady = authMiddleware.RequireRole(identity.RoleAdmin, handler.NewAdminMarkSellerPayoutReadyHandler(txm, markPayoutReady))
		adminPayoutPaid = authMiddleware.RequireRole(identity.RoleAdmin, handler.NewAdminMarkSellerPayoutPaidHandler(txm, markPayoutPaid))
	}

	inner := httpapi.NewRouter(httpapi.Handlers{
		GetBalance:              authMiddleware.Wrap(handler.NewGetBalanceHandler(txm, accounts, createAccount, enableFake)),
		TestTopUp:               testTopUpH,
		GetLedger:               authMiddleware.Wrap(handler.NewGetLedgerHandler(ledgerLister)),
		GetDeposits:             authMiddleware.Wrap(handler.NewGetDepositsHandler(deposits)),
		CreateTopUp:             authMiddleware.Wrap(handler.NewCreateTopUpHandler(txm, createTopUpUC)),
		ListTopUps:              authMiddleware.Wrap(handler.NewListTopUpsHandler(topUps)),
		FakeConfirmTopUp:        fakeTopUpH,
		GetDealInvoice:          authMiddleware.Wrap(handler.NewGetDealInvoiceHandler(dealInvoices)),
		GetDealInvoiceByDeal:    authMiddleware.Wrap(handler.NewGetDealInvoiceByDealHandler(dealInvoices)),
		ListMyDealInvoices:      authMiddleware.Wrap(handler.NewListMyDealInvoicesHandler(dealInvoices)),
		FakeConfirmDealInvoice:  fakeInvoiceH,
		ListMySellerPayouts:     authMiddleware.Wrap(handler.NewListMySellerPayoutsHandler(sellerPayouts)),
		GetSellerPayout:         authMiddleware.Wrap(handler.NewGetSellerPayoutHandler(sellerPayouts)),
		AdminConfirmDealInvoice: adminConfirmInv,
		AdminExpireDealInvoice:  adminExpireInv,
		AdminMarkPayoutReady:    adminPayoutReady,
		AdminMarkPayoutPaid:     adminPayoutPaid,
	}, httplog.Middleware(logger))

	r := http.NewServeMux()
	r.Handle("/billing/", http.StripPrefix("/billing", inner))

	port := dbconfig.EnvOrDefault("BILLING_PORT", "8085")
	logger.Info("http_server_starting", "component", "http.server", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logging.Fatal(logger, "http_server_stopped", "component", "http.server", "error", err)
	}
}
