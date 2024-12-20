package timeutils_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/webitel/webitel-wfm/pkg/timeutils"
)

func TestBetween(t *testing.T) {
	tests := []struct {
		name   string
		curr   time.Time
		from   time.Time
		to     time.Time
		expect bool
	}{
		{
			name:   "current time is equal to from",
			curr:   time.Date(2024, 12, 20, 10, 0, 0, 0, time.UTC),
			from:   time.Date(2024, 12, 20, 10, 0, 0, 0, time.UTC),
			to:     time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC),
			expect: true,
		},
		{
			name:   "current time is equal to to",
			curr:   time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC),
			from:   time.Date(2024, 12, 20, 10, 0, 0, 0, time.UTC),
			to:     time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC),
			expect: true,
		},
		{
			name:   "current time is between from and to",
			curr:   time.Date(2024, 12, 20, 11, 0, 0, 0, time.UTC),
			from:   time.Date(2024, 12, 20, 10, 0, 0, 0, time.UTC),
			to:     time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC),
			expect: true,
		},
		{
			name:   "current time is before from",
			curr:   time.Date(2024, 12, 20, 9, 0, 0, 0, time.UTC),
			from:   time.Date(2024, 12, 20, 10, 0, 0, 0, time.UTC),
			to:     time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC),
			expect: false,
		},
		{
			name:   "current time is after to",
			curr:   time.Date(2024, 12, 20, 13, 0, 0, 0, time.UTC),
			from:   time.Date(2024, 12, 20, 10, 0, 0, 0, time.UTC),
			to:     time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC),
			expect: false,
		},
		{
			name:   "current time is exactly between from and to",
			curr:   time.Date(2024, 12, 20, 11, 59, 59, 0, time.UTC),
			from:   time.Date(2024, 12, 20, 10, 0, 0, 0, time.UTC),
			to:     time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC),
			expect: true,
		},
		{
			name:   "current time is equal to from and to (edge case)",
			curr:   time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC),
			from:   time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC),
			to:     time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC),
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, timeutils.Between(tt.curr, tt.from, tt.to))
		})
	}
}
