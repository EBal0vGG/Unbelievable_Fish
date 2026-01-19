package auction

type AuctionClosed struct {
	AuctionID       string
	WinnerCompanyID string
	FinalPrice      int64
}