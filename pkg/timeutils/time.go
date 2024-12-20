package timeutils

import (
	"time"
)

// Between checks if the current time (curr) falls within the inclusive range
// defined by from and to.
//   - curr is within the range [from, to] = true
//   - curr is exactly equal to from or to = true
func Between(curr time.Time, from time.Time, to time.Time) bool {
	if curr.Equal(from) || (curr.After(from) && curr.Before(to)) {
		return true
	}

	if curr.Equal(to) || (curr.Before(to) && curr.After(from)) {
		return true
	}

	return false
}
