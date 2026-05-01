package main

import (
	"net/http"

	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http/handler"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httplog"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/httpauth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
)

func main() {
	logger := logging.New("deals")
	db, ok := dbconfig.OpenPostgresFromEnv(0)
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()

	uow := dealspg.NewUnitOfWork(db)
	dealRepo := dealspg.NewDealRepository(db)
	confirmationRepo := dealspg.NewDealConfirmationRepository(db)
	projectionRepo := dealspg.NewProjectionRepository(db)
	requestConfirmationUC, err := dealsapp.NewRequestDealConfirmation(uow, dealsapp.NoopConfirmationNotifier{})
	if err != nil {
		logging.Fatal(logger, "request_deal_confirmation_usecase_init_failed", "error", err)
	}
	approveConfirmationUC, err := dealsapp.NewApproveDealConfirmation(uow)
	if err != nil {
		logging.Fatal(logger, "approve_deal_confirmation_usecase_init_failed", "error", err)
	}
	rejectConfirmationUC, err := dealsapp.NewRejectDealConfirmation(uow)
	if err != nil {
		logging.Fatal(logger, "reject_deal_confirmation_usecase_init_failed", "error", err)
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
	requestShipmentUC, err := dealsapp.NewRequestShipment(uow)
	if err != nil {
		logging.Fatal(logger, "request_shipment_usecase_init_failed", "error", err)
	}
	updatePriceUC, err := dealsapp.NewUpdateDealPrice(uow)
	if err != nil {
		logging.Fatal(logger, "update_deal_price_usecase_init_failed", "error", err)
	}
	tokenProvider := identityauth.NewTokenProvider(
		dbconfig.EnvOrDefault("IDENTITY_TOKEN_SECRET", "dev-secret"),
		dbconfig.EnvDurationMinutes("IDENTITY_TOKEN_TTL_MINUTES", 24*60),
	)

	authMiddleware := identityauth.NewMiddleware(tokenProvider, httpauth.JSONErrorHandler("deals_auth_error"))
	router := httpapi.NewRouter(httpapi.Handlers{
		GetDealProjection:   handler.NewGetProjectionByAuctionIDHandler(dealsapp.NewGetProjectionByAuctionID(projectionRepo)),
		GetDealByAuction:    handler.NewGetDealByAuctionIDHandler(dealsapp.NewGetDealByAuctionID(dealRepo)),
		GetDeal:             handler.NewGetDealByIDHandler(dealsapp.NewGetDealByID(dealRepo)),
		GetConfirmations:    handler.NewGetDealConfirmationsHandler(dealsapp.NewGetDealConfirmations(dealRepo, confirmationRepo)),
		RequestConfirmation: handler.NewRequestDealConfirmationHandler(requestConfirmationUC),
		ApproveConfirmation: handler.NewApproveDealConfirmationHandler(approveConfirmationUC),
		RejectConfirmation:  handler.NewRejectDealConfirmationHandler(rejectConfirmationUC),
		PrepareContract:     handler.NewPrepareContractHandler(prepareUC),
		SignContract:        handler.NewSignContractHandler(signUC),
		RequestPayment:      handler.NewRequestPaymentHandler(requestPaymentUC),
		RequestShipment:     handler.NewRequestShipmentHandler(requestShipmentUC),
		UpdateDealPrice:     handler.NewUpdateDealPriceHandler(updatePriceUC),
	}, httplog.Middleware(logger), authMiddleware.Wrap)

	port := dbconfig.EnvOrDefault("DEALS_PORT", "8083")
	logger.Info("http_server_starting", "component", "http.server", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logging.Fatal(logger, "http_server_stopped", "component", "http.server", "error", err)
	}
}
