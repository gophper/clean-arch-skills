package system

import "time"

// Clock is an adapter that returns the real wall-clock time.
// In tests, swap it with a fake that returns a fixed time.
type Clock struct{}

func NewClock() Clock        { return Clock{} }
func (Clock) Now() time.Time { return time.Now() }
