package typeutil

import "time"

// Now returns the current time in UTC (UTC+0).
// Use this instead of time.Now() to ensure all timestamps are UTC.
func Now() time.Time {
	return time.Now().UTC()
}
