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

func NewTimestamp(timestamp int64) pgtype.Timestamp {
	if timestamp <= 0 {
		return pgtype.Timestamp{}
	}

	return pgtype.Timestamp{
		Time:  time.Unix(timestamp, 0),
		Valid: true,
	}
}
