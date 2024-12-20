package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestElementsMatch(t *testing.T) {
	tests := []struct {
		name   string
		x      []int64
		y      []int64
		expect bool
	}{
		{
			name:   "equal slices",
			x:      []int64{1, 2, 3, 4},
			y:      []int64{4, 3, 2, 1},
			expect: true,
		},
		{
			name:   "slices with different lengths",
			x:      []int64{1, 2, 3},
			y:      []int64{1, 2},
			expect: false,
		},
		{
			name:   "different elements",
			x:      []int64{1, 2, 3},
			y:      []int64{4, 5, 6},
			expect: false,
		},
		{
			name:   "different frequencies",
			x:      []int64{1, 2, 2, 3},
			y:      []int64{1, 2, 3, 3},
			expect: false,
		},
		{
			name:   "empty slices",
			x:      []int64{},
			y:      []int64{},
			expect: true,
		},
		{
			name:   "nne empty slice",
			x:      []int64{},
			y:      []int64{1, 2, 3},
			expect: false,
		},
		{
			name:   "empty slice with matching elements",
			x:      []int64{1, 2, 3},
			y:      []int64{1, 2, 3},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, ElementsMatch(tt.x, tt.y))
		})
	}
}
