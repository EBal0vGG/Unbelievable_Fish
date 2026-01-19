package auction

import "errors"

var (
	ErrAuctionCannotBePublished = errors.New("auction cannot be published")
	ErrAuctionNotActive         = errors.New("auction not active")
	ErrCannotCloseAuction       = errors.New("cannot close auction")
	ErrInvalidStateTransition   = errors.New("invalid auction state transition")
	ErrNoBids                   = errors.New("no bids")
	ErrCannotCancelWithBids     = errors.New("cannot cancel auction with bids")
	ErrBidderCompanyIDEmpty     = errors.New("bidder company id is empty")
	ErrBidAmountNonPositive     = errors.New("bid amount must be positive")
	ErrBidPlacedAtZero          = errors.New("bid placed at time is zero")
)