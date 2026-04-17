package usecase

import "time"

// SystemClock returns wall-clock UTC time.
type SystemClock struct{}

// Now returns the current UTC timestamp.
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}
