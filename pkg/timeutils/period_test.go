package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPeriod(t *testing.T) {
	type args struct {
		startDate    time.Time
		endDate      time.Time
		boundaryType boundaryType
	}

	tests := []struct {
		name string
		args args
		want Period
	}{
		{
			name: "NewPeriod_WithStartDateBeforeEndDate",
			args: args{
				startDate:    time.Date(2023, 1, 2, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeAll,
			},
			want: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 2, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeAll,
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := NewPeriod(tt.args.startDate, tt.args.endDate, tt.args.boundaryType)
				assert.Equal(t, tt.want, got)
			},
		)
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		period Period
		other  Period
		want   bool
	}{
		{
			name: "Contains_WithContainedPeriod",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			other: Period{
				startDate:    time.Date(2023, 1, 2, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			want: true,
		},
		{
			name: "Contains_WithNonContainedPeriod",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 2, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			other: Period{
				startDate:    time.Date(2023, 1, 2, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := tt.period.Contains(tt.other)
				assert.Equal(t, tt.want, got)
			},
		)
	}
}

func TestContainsInterval(t *testing.T) {
	tests := []struct {
		name   string
		period Period
		other  Period
		want   bool
	}{
		{
			name: "ContainsInterval_WithContainedPeriod",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			other: Period{
				startDate:    time.Date(2023, 1, 2, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			want: true,
		},
		{
			name: "ContainsInterval_WithFullContainedPeriod",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 5, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			other: Period{
				startDate:    time.Date(2023, 1, 2, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			want: true,
		},
		{
			name: "ContainsInterval_WithNonContainedPeriod",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			other: Period{
				startDate:    time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 4, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			want: false,
		},
		{
			name: "ContainsInterval_WithSamePeriods",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			other: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			want: true,
		},
		{
			name: "ContainsInterval_WithSameStartDateAndSameBoundaryFullContainedPeriod",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 9, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			other: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := tt.period.containsInterval(tt.other)
				assert.Equal(t, tt.want, got)
			},
		)
	}
}

func TestContainsDatePoint(t *testing.T) {
	tests := []struct {
		name         string
		period       Period
		datePoint    time.Time
		boundaryType boundaryType
		want         bool
	}{
		{
			name: "ContainsDatePoint_WithExcludeAllBoundaryType",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: ExcludeAll,
			},
			datePoint:    time.Date(2023, 1, 2, 0, 0, 0, 0, time.Local),
			boundaryType: ExcludeAll,
			want:         true,
		},
		{
			name: "ContainsDatePoint_WithIncludeAllBoundaryType",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeAll,
			},
			datePoint:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			boundaryType: IncludeAll,
			want:         true,
		},
		{
			name: "ContainsDatePoint_WithExcludeStartIncludeEndBoundaryType",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: ExcludeStartIncludeEnd,
			},
			datePoint:    time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
			boundaryType: ExcludeStartIncludeEnd,
			want:         true,
		},
		{
			name: "ContainsDatePoint_WithIncludeStartExcludeEndBoundaryType",
			period: Period{
				startDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
				endDate:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.Local),
				boundaryType: IncludeStartExcludeEnd,
			},
			datePoint:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			boundaryType: IncludeStartExcludeEnd,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := tt.period.containsDatePoint(tt.datePoint, tt.boundaryType)
				assert.Equal(t, tt.want, got)
			},
		)
	}
}
