package handler

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/internal/model"
)

// ErrDate: the validator lets "YYYY-MM-DD" through by pattern, so a day that does not
// exist, such as 2026-02-30, is caught here.
var ErrDate = errors.InvalidArgument("invalid input: a date should be a calendar day, YYYY-MM-DD", errors.WithID("handler.date"))

func parseDate(field, s string) (pgtype.Date, error) {
	d, err := model.ParseDate(s)
	if err != nil {
		return pgtype.Date{}, errors.Wrap(ErrDate, errors.WithCause(err), errors.WithValue("field", field))
	}

	return d, nil
}
