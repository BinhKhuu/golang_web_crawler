package typeutil

import "time"

// UTCTimeNow returns the current time in UTC (UTC+0).
// Use this instead of time.UTCTimeNow() to ensure all timestamps are UTC.
func UTCTimeNow() time.Time {
	return time.Now().UTC()
}
