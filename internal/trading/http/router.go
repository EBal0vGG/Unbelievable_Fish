package httpapi

import (
	"net/http"
	"strings"
)

type Router struct {
	placeBid http.Handler
}

func NewRouter(
	placeBid http.Handler,
) *Router {
	return &Router{
		placeBid: placeBid,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/auctions/") && strings.HasSuffix(req.URL.Path, "/bids") {
		r.placeBid.ServeHTTP(w, req)
		return
	}
	http.NotFound(w, req)
}
