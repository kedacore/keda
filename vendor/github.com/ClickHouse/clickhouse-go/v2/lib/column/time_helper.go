package column

import "time"

// getTimeWithDifferentLocation returns the same time but with different location, e.g.
// "2024-08-15 13:22:34 -03:00" will become "2024-08-15 13:22:34 +04:00".
func getTimeWithDifferentLocation(t time.Time, loc *time.Location) time.Time {
	year, month, day := t.Date()
	hour, minute, sec := t.Clock()

	return time.Date(year, month, day, hour, minute, sec, t.Nanosecond(), loc)
}
