package app

import "time"

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}
