package httpapi

import (
	"net/http"
	"strings"
)

type Router struct {
	getProjection       http.Handler
	getDeal             http.Handler
	getDealByAuction    http.Handler
	getConfirmations    http.Handler
	requestConfirmation http.Handler
	approveConfirmation http.Handler
	rejectConfirmation  http.Handler
	prepareContract     http.Handler
	signContract        http.Handler
	requestPayment      http.Handler
	requestShipment     http.Handler
	updateDealPrice     http.Handler
}

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
) *Router {
	return &Router{
		getProjection:       getProjection,
		getDeal:             getDeal,
		getDealByAuction:    getDealByAuction,
		getConfirmations:    getConfirmations,
		requestConfirmation: requestConfirmation,
		approveConfirmation: approveConfirmation,
		rejectConfirmation:  rejectConfirmation,
		prepareContract:     prepareContract,
		signContract:        signContract,
		requestPayment:      requestPayment,
		requestShipment:     requestShipment,
		updateDealPrice:     updateDealPrice,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/deal-projections/") {
		r.getProjection.ServeHTTP(w, req)
		return
	}
	if req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/deals/by-auction/") {
		r.getDealByAuction.ServeHTTP(w, req)
		return
	}
	if strings.HasPrefix(req.URL.Path, "/deals/") {
		switch {
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/confirmations"):
			r.getConfirmations.ServeHTTP(w, req)
			return
		case req.Method == http.MethodGet:
			r.getDeal.ServeHTTP(w, req)
			return
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/confirmations"):
			r.requestConfirmation.ServeHTTP(w, req)
			return
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/approve"):
			r.approveConfirmation.ServeHTTP(w, req)
			return
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/reject"):
			r.rejectConfirmation.ServeHTTP(w, req)
			return
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/contract/prepare"):
			r.prepareContract.ServeHTTP(w, req)
			return
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/contract/sign"):
			r.signContract.ServeHTTP(w, req)
			return
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/payment/request"):
			r.requestPayment.ServeHTTP(w, req)
			return
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/shipment/request"):
			r.requestShipment.ServeHTTP(w, req)
			return
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/price"):
			r.updateDealPrice.ServeHTTP(w, req)
			return
		}
	}
	http.NotFound(w, req)
}
