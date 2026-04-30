package httpapi

import (
	"net/http"
	"strings"
)

type Router struct {
	publishAuction http.Handler
	placeBid       http.Handler
	closeAuction   http.Handler
	cancelAuction  http.Handler
	getByID        http.Handler
	getByLot       http.Handler
}

func NewRouter(
	publishAuction http.Handler,
	placeBid http.Handler,
	closeAuction http.Handler,
	cancelAuction http.Handler,
	getByID http.Handler,
	getByLot http.Handler,
) *Router {
	return &Router{
		publishAuction: publishAuction,
		placeBid:       placeBid,
		closeAuction:   closeAuction,
		cancelAuction:  cancelAuction,
		getByID:        getByID,
		getByLot:       getByLot,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/auctions/") && strings.HasSuffix(req.URL.Path, "/publish") {
		r.publishAuction.ServeHTTP(w, req)
		return
	}
	if req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/auctions/") && strings.HasSuffix(req.URL.Path, "/bids") {
		r.placeBid.ServeHTTP(w, req)
		return
	}
	if req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/auctions/") && strings.HasSuffix(req.URL.Path, "/close") {
		r.closeAuction.ServeHTTP(w, req)
		return
	}
	if req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/auctions/") && strings.HasSuffix(req.URL.Path, "/cancel") {
		r.cancelAuction.ServeHTTP(w, req)
		return
	}
	if req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/auctions/by-lot/") {
		r.getByLot.ServeHTTP(w, req)
		return
	}
	if req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/auctions/") {
		r.getByID.ServeHTTP(w, req)
		return
	}
	http.NotFound(w, req)
}
