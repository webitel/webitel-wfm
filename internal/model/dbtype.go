package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func NewDate(timestamp int64) pgtype.Date {
	if timestamp <= 0 {
		return pgtype.Date{}
	}

	return pgtype.Date{
		Time:  time.Unix(timestamp, 0),
		Valid: true,
	}
}

// DateFromMillis reads a calendar date of the API: 00:00 UTC of the day in epoch
// milliseconds, which the validator enforces; 0 is no date.
func DateFromMillis(ms int64) pgtype.Date {
	if ms <= 0 {
		return pgtype.Date{}
	}

	t := time.UnixMilli(ms).UTC()

	return pgtype.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

// DateMillis writes a calendar date for the API as 00:00 UTC of the day in epoch
// milliseconds, whatever location the date was read in; no date is 0.
func DateMillis(d pgtype.Date) int64 {
	if !d.Valid {
		return 0
	}

	return time.Date(d.Time.Year(), d.Time.Month(), d.Time.Day(), 0, 0, 0, 0, time.UTC).UnixMilli()
}

// TimestampFromMillis reads a moment of the API, epoch milliseconds; 0 is none.
func TimestampFromMillis(ms int64) pgtype.Timestamp {
	if ms <= 0 {
		return pgtype.Timestamp{}
	}

	return pgtype.Timestamp{Time: time.UnixMilli(ms).UTC(), Valid: true}
}

// DayStart is 00:00 of a date as a timestamp in UTC. pgx sends a timestamp by its wall
// clock, so one in the server's local zone would move the day west of UTC.
func DayStart(d pgtype.Date) pgtype.Timestamp {
	if !d.Valid {
		return pgtype.Timestamp{}
	}

	return pgtype.Timestamp{Time: time.Date(d.Time.Year(), d.Time.Month(), d.Time.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}
