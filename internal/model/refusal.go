package model

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// Violation is one broken business rule: which operator, on which day, and why.
type Violation struct {
	OperatorID int64
	Date       pgtype.Date
	Reason     string // a reason id, <layer>.<entity>.<what>
}

type Violations []Violation

// String is what the error interceptor prints into the detail of a refusal.
func (vs Violations) String() string {
	parts := make([]string, 0, len(vs))
	for _, v := range vs {
		parts = append(parts, fmt.Sprintf("operator %d on %s: %s", v.OperatorID, v.Date.Time.UTC().Format(time.DateOnly), v.Reason))
	}

	return strings.Join(parts, ", ")
}

const violationsKey = "violations"

// NewRefusal is a business refusal: FailedPrecondition with an id of its own.
func NewRefusal(id, msg string) error {
	return errors.New(msg, errors.WithCode(codes.FailedPrecondition), errors.WithID(id))
}

// Refuse attaches violations to a refusal. It must stay the outermost wrapper: a WithCause on top
// hides the list from errors.Values, and so from the client's detail.
func Refuse(refusal error, vs ...Violation) error {
	if len(vs) == 0 {
		return refusal
	}

	return errors.Wrap(refusal, errors.WithValue(violationsKey, Violations(slices.Clone(vs))))
}

func ViolationsOf(err error) Violations {
	if vs, ok := errors.Value(err, violationsKey).(Violations); ok {
		return vs
	}

	return nil
}
