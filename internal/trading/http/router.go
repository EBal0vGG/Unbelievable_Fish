package httpapi

import (
	"net/http"
	"strings"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http/handler"
)

type Router struct {
	create *handler.CreateAuctionHandler
	publish *handler.PublishAuctionHandler
	placeBid *handler.PlaceBidHandler
	closeAuction *handler.CloseAuctionHandler
	cancel *handler.CancelAuctionHandler
}

func NewRouter(
	create *app.CreateAuction,
	publish *app.PublishAuction,
	placeBid *app.PlaceBid,
	closeAuction *app.CloseAuction,
	cancel *app.CancelAuction,
) *Router {
	return &Router{
		create:       handler.NewCreateAuctionHandler(create),
		publish:      handler.NewPublishAuctionHandler(publish),
		placeBid:     handler.NewPlaceBidHandler(placeBid),
		closeAuction: handler.NewCloseAuctionHandler(closeAuction),
		cancel:       handler.NewCancelAuctionHandler(cancel),
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
