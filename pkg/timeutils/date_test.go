package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDate(t *testing.T) {
	tests := []struct {
		input    time.Time
		expected time.Time
	}{
		{
			input:    time.Date(2025, 1, 8, 14, 30, 0, 0, time.UTC),
			expected: time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2025, 12, 25, 23, 59, 59, 999999999, time.Local),
			expected: time.Date(2025, 12, 25, 0, 0, 0, 0, time.Local),
		},
		{
			input:    time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, Date(tt.input))
		})
	}
}
