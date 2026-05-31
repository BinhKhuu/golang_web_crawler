package typeutil

import "time"

func FloatPtr(f float64) *float64 { return new(f) }

func StringPtr(s string) *string { return new(s) }

func TimePtr(t time.Time) *time.Time { return new(t) }
