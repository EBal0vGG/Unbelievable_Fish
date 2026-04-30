package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(
	publishAuction http.Handler,
	placeBid http.Handler,
	closeAuction http.Handler,
	cancelAuction http.Handler,
	getByID http.Handler,
	getByLot http.Handler,
	middlewares ...func(http.Handler) http.Handler,
) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares...)
	r.Method(http.MethodPost, "/auctions/{auctionID}/publish", publishAuction)
	r.Method(http.MethodPost, "/auctions/{auctionID}/bids", placeBid)
	r.Method(http.MethodPost, "/auctions/{auctionID}/close", closeAuction)
	r.Method(http.MethodPost, "/auctions/{auctionID}/cancel", cancelAuction)
	r.Method(http.MethodGet, "/auctions/by-lot/{lotID}", getByLot)
	r.Method(http.MethodGet, "/auctions/{auctionID}", getByID)
	return r
}
