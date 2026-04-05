package catalog

import "time"

type Instant struct {
	time time.Time
}

func NewInstant(value time.Time) Instant {
	return Instant{time: value.UTC()}
}

func (i Instant) Time() time.Time {
	return i.time
}

type AuctionSchedule struct {
	startsAt Instant
	duration time.Duration
}

func NewAuctionScheduleAt(startsAt time.Time, duration time.Duration) *AuctionSchedule {
	return &AuctionSchedule{
		startsAt: NewInstant(startsAt),
		duration: duration,
	}
}

func (s *AuctionSchedule) IsValid() bool {
	if s == nil {
		return false
	}
	return !s.startsAt.time.IsZero() && s.duration > 0
}

func (s *AuctionSchedule) StartsAt() time.Time {
	return s.startsAt.time
}

func (s *AuctionSchedule) EndsAt() time.Time {
	return s.startsAt.time.Add(s.duration)
}

func (s *AuctionSchedule) Duration() time.Duration {
	return s.duration
}
