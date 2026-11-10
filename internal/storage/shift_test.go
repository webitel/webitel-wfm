package storage

import (
	"context"
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/internal/model"
)

func TestValidateShifts(t *testing.T) {
	cases := []struct {
		name string
		in   []*model.Shift
		want error
	}{
		{
			name: "a day shift with pauses back to back, the later one first",
			in: []*model.Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02",
				"2026-11-02 13:00+02", "2026-11-02 13:30+02", "2026-11-02 12:00+02", "2026-11-02 13:00+02")},
		},
		{
			name: "a shift of 24 hours",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-02 08:00+02", "2026-11-03 08:00+02")},
		},
		{
			name: "a shift that ends as it starts",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-02 08:00+02", "2026-11-02 08:00+02")},
			want: ErrShiftDuration,
		},
		{
			name: "a shift over the night the clocks go back",
			in:   []*model.Shift{shiftAt(t, 1, "2026-10-24 22:00+03", "2026-10-25 22:00+02")},
			want: ErrShiftDuration,
		},
		{
			name: "a pause past the end of the shift",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 16:30+02", "2026-11-02 17:30+02")},
			want: ErrShiftPauseOutside,
		},
		{
			name: "a pause before the start of the shift",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 08:30+02", "2026-11-02 09:30+02")},
			want: ErrShiftPauseOutside,
		},
		{
			name: "a pause that ends as it starts",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 12:00+02", "2026-11-02 12:00+02")},
			want: ErrShiftPauseOutside,
		},
		{
			name: "pauses over each other, the later one first",
			in: []*model.Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02",
				"2026-11-02 12:30+02", "2026-11-02 13:30+02", "2026-11-02 12:00+02", "2026-11-02 13:00+02")},
			want: ErrShiftPauseOverlap,
		},
		{
			name: "shifts of an agent over each other, the later one first",
			in: []*model.Shift{
				shiftAt(t, 1, "2026-11-02 16:00+02", "2026-11-02 20:00+02"),
				shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
			},
			want: ErrShiftRequestOverlap,
		},
		{
			name: "shifts of an agent over each other, another agent's in between",
			in: []*model.Shift{
				shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
				shiftAt(t, 2, "2026-11-02 10:00+02", "2026-11-02 11:00+02"),
				shiftAt(t, 1, "2026-11-02 16:00+02", "2026-11-02 20:00+02"),
			},
			want: ErrShiftRequestOverlap,
		},
		{
			name: "shifts of an agent back to back, the later one first, another agent at the same time",
			in: []*model.Shift{
				shiftAt(t, 1, "2026-11-02 17:00+02", "2026-11-02 21:00+02"),
				shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
				shiftAt(t, 2, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateShifts(tc.in); !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// A request wrong on its own is turned away before the transaction: this storage has
// no database to reach.
func TestCreateShiftsValidatesBeforeTheTransaction(t *testing.T) {
	in := []*model.Shift{shiftAt(t, 1, "2026-11-02 08:00+02", "2026-11-02 08:00+02")}

	if _, err := (&AgentWorkingSchedule{}).CreateShifts(context.Background(), &model.SignedInUser{ID: 1, DomainID: 1}, 1, in); !errors.Is(err, ErrShiftDuration) {
		t.Errorf("err = %v, want %v", err, ErrShiftDuration)
	}
}
