package model

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// A date travels as 00:00 UTC of its day in milliseconds, whatever location it was read in.
func TestDateMillis(t *testing.T) {
	kyiv, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		t.Fatal(err)
	}

	// West of UTC a local reading of 00:00 UTC falls on the day before.
	local := time.Local
	time.Local = time.FixedZone("UTC-5", -5*60*60)

	t.Cleanup(func() { time.Local = local })

	utc := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC).UnixMilli()

	if got := DateMillis(pgtype.Date{Time: time.Date(2026, 11, 20, 0, 0, 0, 0, kyiv), Valid: true}); got != utc {
		t.Errorf("a date read in Kyiv goes out as %d, want %d", got, utc)
	}

	if d := DateFromMillis(utc); !d.Valid || d.Time.Day() != 20 || d.Time.Location() != time.UTC {
		t.Errorf("%d comes in as %+v, want 2026-11-20", utc, d)
	}

	if DateFromMillis(0).Valid || DateMillis(pgtype.Date{}) != 0 {
		t.Error("0 must be no date both ways")
	}
}
