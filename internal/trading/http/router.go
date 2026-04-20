package httpapi

import (
	"net/http"
	"strings"
)

type Router struct {
	createAuction  http.Handler
	publishAuction http.Handler
	placeBid       http.Handler
	closeAuction   http.Handler
	cancelAuction  http.Handler
}

func NewRouter(
	createAuction http.Handler,
	publishAuction http.Handler,
	placeBid http.Handler,
	closeAuction http.Handler,
	cancelAuction http.Handler,
) *Router {
	return &Router{
		createAuction:  createAuction,
		publishAuction: publishAuction,
		placeBid:       placeBid,
		closeAuction:   closeAuction,
		cancelAuction:  cancelAuction,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost && req.URL.Path == "/auctions" {
		r.createAuction.ServeHTTP(w, req)
		return
	}
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
	http.NotFound(w, req)
}
