package auction

func determineWinner(bids []Bid) (Bid, bool) {
	if len(bids) == 0 {
		return Bid{}, false
	}
	winner := bids[0]
	for _, bid := range bids[1:] {
		if bid.Amount() > winner.Amount() {
			winner = bid
			continue
		}
		if bid.Amount() == winner.Amount() && bid.PlacedAt().Before(winner.PlacedAt()) {
			winner = bid
		}
	}
	return winner, true
}
