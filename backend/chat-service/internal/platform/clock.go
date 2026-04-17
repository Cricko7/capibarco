package platform

import "time"

// SystemClock returns current UTC time.
type SystemClock struct{}

// Now returns current UTC time.
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}
