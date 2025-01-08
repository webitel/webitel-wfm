package timeutils

import "time"

// Date extracts only the date part (without the time).
func Date(t time.Time) time.Time {
	year, month, day := t.Date()

	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}
