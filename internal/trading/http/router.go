package httpapi

import (
	"net/http"
	"strings"
)

type Router struct {
	create       http.Handler
	publish      http.Handler
	placeBid     http.Handler
	closeAuction http.Handler
	cancel       http.Handler
}

func NewRouter(
	create http.Handler,
	publish http.Handler,
	placeBid http.Handler,
	closeAuction http.Handler,
	cancel http.Handler,
) *Router {
	return &Router{
		create:       create,
		publish:      publish,
		placeBid:     placeBid,
		closeAuction: closeAuction,
		cancel:       cancel,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost && req.URL.Path == "/auctions" {
		r.create.ServeHTTP(w, req)
		return
	}
	if req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/auctions/") {
		switch {
		case strings.HasSuffix(req.URL.Path, "/publish"):
			r.publish.ServeHTTP(w, req)
			return
		case strings.HasSuffix(req.URL.Path, "/bids"):
			r.placeBid.ServeHTTP(w, req)
			return
		case strings.HasSuffix(req.URL.Path, "/close"):
			r.closeAuction.ServeHTTP(w, req)
			return
		case strings.HasSuffix(req.URL.Path, "/cancel"):
			r.cancel.ServeHTTP(w, req)
			return
		}
	}
	http.NotFound(w, req)
}
