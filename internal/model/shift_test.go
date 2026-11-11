package model

import (
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

func shiftAt(t *testing.T, agent int64, start, end string, pauses ...string) *Shift {
	t.Helper()

	at := func(s string) time.Time {
		ts, err := time.Parse("2006-01-02 15:04-07", s)
		if err != nil {
			t.Fatal(err)
		}

		return ts
	}

	shift := &Shift{AgentID: agent, Start: at(start), End: at(end)}
	for i := 0; i+1 < len(pauses); i += 2 {
		shift.Pauses = append(shift.Pauses, &ShiftPause{Start: at(pauses[i]), End: at(pauses[i+1])})
	}

	return shift
}

func TestValidateShifts(t *testing.T) {
	cases := []struct {
		name string
		in   []*Shift
		want error
	}{
		{
			name: "a day shift with pauses back to back, the later one first",
			in: []*Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02",
				"2026-11-02 13:00+02", "2026-11-02 13:30+02", "2026-11-02 12:00+02", "2026-11-02 13:00+02")},
		},
		{
			name: "a shift of 24 hours",
			in:   []*Shift{shiftAt(t, 1, "2026-11-02 08:00+02", "2026-11-03 08:00+02")},
		},
		{
			name: "a shift that ends as it starts",
			in:   []*Shift{shiftAt(t, 1, "2026-11-02 08:00+02", "2026-11-02 08:00+02")},
			want: ErrShiftDuration,
		},
		{
			name: "a shift over the night the clocks go back",
			in:   []*Shift{shiftAt(t, 1, "2026-10-24 22:00+03", "2026-10-25 22:00+02")},
			want: ErrShiftDuration,
		},
		{
			name: "a pause past the end of the shift",
			in:   []*Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 16:30+02", "2026-11-02 17:30+02")},
			want: ErrShiftPauseOutside,
		},
		{
			name: "a pause before the start of the shift",
			in:   []*Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 08:30+02", "2026-11-02 09:30+02")},
			want: ErrShiftPauseOutside,
		},
		{
			name: "a pause that ends as it starts",
			in:   []*Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 12:00+02", "2026-11-02 12:00+02")},
			want: ErrShiftPauseOutside,
		},
		{
			name: "pauses over each other, the later one first",
			in: []*Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02",
				"2026-11-02 12:30+02", "2026-11-02 13:30+02", "2026-11-02 12:00+02", "2026-11-02 13:00+02")},
			want: ErrShiftPauseOverlap,
		},
		{
			name: "shifts of an agent over each other, the later one first",
			in: []*Shift{
				shiftAt(t, 1, "2026-11-02 16:00+02", "2026-11-02 20:00+02"),
				shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
			},
			want: ErrShiftRequestOverlap,
		},
		{
			name: "shifts of an agent over each other, another agent's in between",
			in: []*Shift{
				shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
				shiftAt(t, 2, "2026-11-02 10:00+02", "2026-11-02 11:00+02"),
				shiftAt(t, 1, "2026-11-02 16:00+02", "2026-11-02 20:00+02"),
			},
			want: ErrShiftRequestOverlap,
		},
		{
			name: "shifts of an agent back to back, the later one first, another agent at the same time",
			in: []*Shift{
				shiftAt(t, 1, "2026-11-02 17:00+02", "2026-11-02 21:00+02"),
				shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
				shiftAt(t, 2, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateShifts(tc.in); !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}
