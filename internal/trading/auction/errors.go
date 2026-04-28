package auction

import "errors"

var (
	ErrAuctionCannotBePublished = errors.New("auction cannot be published")
	ErrAuctionNotActive         = errors.New("auction not active")
	ErrCannotCloseAuction       = errors.New("cannot close auction")
	ErrInvalidStateTransition   = errors.New("invalid auction state transition")
	ErrNoBids                   = errors.New("no bids")
	ErrCannotCancelWithBids     = errors.New("cannot cancel auction with bids")
	ErrAuctionNotStarted        = errors.New("auction not started")
	ErrAuctionAlreadyEnded      = errors.New("auction already ended")
	ErrInvalidSchedule          = errors.New("invalid auction schedule")
	ErrBidTooLow                = errors.New("bid amount must be greater than current price")
	ErrBidStepTooSmall          = errors.New("bid amount is below minimum bid step")
	ErrAuctionIDEmpty           = errors.New("auction id is empty")
	ErrLotIDEmpty               = errors.New("lot id is empty")
	ErrInvalidStartPrice        = errors.New("invalid start price")
	ErrInvalidMinBidStep        = errors.New("invalid minimum bid step")
	ErrBidderCompanyIDEmpty     = errors.New("bidder company id is empty")
	ErrBidAmountNonPositive     = errors.New("bid amount must be positive")
	ErrBidPlacedAtZero          = errors.New("bid placed at time is zero")
)
