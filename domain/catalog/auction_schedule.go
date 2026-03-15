package catalog

import "time"

func NewAuctionScheduleAt(startsAt time.Time) *AuctionSchedule {
	return &AuctionSchedule{startsAt: NewInstant(startsAt)}
}
