package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(
	getProjection http.Handler,
	getDeal http.Handler,
	getDealByAuction http.Handler,
	getConfirmations http.Handler,
	requestConfirmation http.Handler,
	approveConfirmation http.Handler,
	rejectConfirmation http.Handler,
	prepareContract http.Handler,
	signContract http.Handler,
	requestPayment http.Handler,
	requestShipment http.Handler,
	updateDealPrice http.Handler,
	middlewares ...func(http.Handler) http.Handler,
) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares...)
	r.Method(http.MethodGet, "/deal-projections/{auctionID}", getProjection)
	r.Method(http.MethodGet, "/deals/by-auction/{auctionID}", getDealByAuction)
	r.Method(http.MethodGet, "/deals/{dealID}/confirmations", getConfirmations)
	r.Method(http.MethodGet, "/deals/{dealID}", getDeal)
	r.Method(http.MethodPost, "/deals/{dealID}/confirmations", requestConfirmation)
	r.Method(http.MethodPost, "/deals/{dealID}/confirmations/{confirmationID}/approve", approveConfirmation)
	r.Method(http.MethodPost, "/deals/{dealID}/confirmations/{confirmationID}/reject", rejectConfirmation)
	r.Method(http.MethodPost, "/deals/{dealID}/contract/prepare", prepareContract)
	r.Method(http.MethodPost, "/deals/{dealID}/contract/sign", signContract)
	r.Method(http.MethodPost, "/deals/{dealID}/payment/request", requestPayment)
	r.Method(http.MethodPost, "/deals/{dealID}/shipment/request", requestShipment)
	r.Method(http.MethodPost, "/deals/{dealID}/price", updateDealPrice)
	return r
}
