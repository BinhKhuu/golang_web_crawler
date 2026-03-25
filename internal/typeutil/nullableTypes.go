package typeutil

import "time"

func FloatPtr(f float64) *float64    { return &f }
func StringPtr(s string) *string     { return &s }
func TimePtr(t time.Time) *time.Time { return &t }
