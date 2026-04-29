package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(
	getProjection http.Handler,
	getDeal http.Handler,
	getDealByAuction http.Handler,
	confirmDeal http.Handler,
	prepareContract http.Handler,
	signContract http.Handler,
	requestPayment http.Handler,
	markDealPaid http.Handler,
	requestShipment http.Handler,
	markDealShipped http.Handler,
	completeDeal http.Handler,
	cancelDeal http.Handler,
	updateDealPrice http.Handler,
	middlewares ...func(http.Handler) http.Handler,
) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares...)
	r.Method(http.MethodGet, "/deal-projections/{auctionID}", getProjection)
	r.Method(http.MethodGet, "/deals/by-auction/{auctionID}", getDealByAuction)
	r.Method(http.MethodGet, "/deals/{dealID}", getDeal)
	r.Method(http.MethodPost, "/deals/{dealID}/confirm", confirmDeal)
	r.Method(http.MethodPost, "/deals/{dealID}/contract/prepare", prepareContract)
	r.Method(http.MethodPost, "/deals/{dealID}/contract/sign", signContract)
	r.Method(http.MethodPost, "/deals/{dealID}/payment/request", requestPayment)
	r.Method(http.MethodPost, "/deals/{dealID}/payment/mark-paid", markDealPaid)
	r.Method(http.MethodPost, "/deals/{dealID}/shipment/request", requestShipment)
	r.Method(http.MethodPost, "/deals/{dealID}/shipment/mark-shipped", markDealShipped)
	r.Method(http.MethodPost, "/deals/{dealID}/complete", completeDeal)
	r.Method(http.MethodPost, "/deals/{dealID}/cancel", cancelDeal)
	r.Method(http.MethodPost, "/deals/{dealID}/price", updateDealPrice)
	return r
}
