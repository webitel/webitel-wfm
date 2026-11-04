package model

import (
	"slices"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// Violation is one broken business rule: which operator, on which day, and why.
type Violation struct {
	OperatorID int64       `json:"operator_id"`
	Date       pgtype.Date `json:"date"`
	Reason     string      `json:"reason"` // a reason id, <layer>.<entity>.<what>
}

type refusalKey string

// Not a plain string, so the error interceptor keeps the list out of the client's detail; the log still gets it.
const violationsKey refusalKey = "violations"

// Refuse attaches violations to a refusal: an errors.FailedPrecondition sentinel with its own id.
func Refuse(refusal error, vs ...Violation) error {
	if len(vs) == 0 {
		return refusal
	}

	return errors.Wrap(refusal, errors.WithValue(violationsKey, slices.Clone(vs)))
}

func ViolationsOf(err error) []Violation {
	if vs, ok := errors.Value(err, violationsKey).([]Violation); ok {
		return vs
	}

	return nil
}
