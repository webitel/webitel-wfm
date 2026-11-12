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

// ParseDate reads a calendar date of the API, "YYYY-MM-DD"; an empty string is no date.
func ParseDate(s string) (pgtype.Date, error) {
	if s == "" {
		return pgtype.Date{}, nil
	}

	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return pgtype.Date{}, err
	}

	return pgtype.Date{Time: t, Valid: true}, nil
}

// FormatDate writes a calendar date for the API, "YYYY-MM-DD"; no date is an empty string.
func FormatDate(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}

	return d.Time.Format(time.DateOnly)
}

func NewTimestamp(timestamp int64) pgtype.Timestamp {
	if timestamp <= 0 {
		return pgtype.Timestamp{}
	}

	return pgtype.Timestamp{
		Time:  time.Unix(timestamp, 0),
		Valid: true,
	}
}
