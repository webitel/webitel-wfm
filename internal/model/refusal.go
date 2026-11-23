package model

import (
	"slices"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

// Violation is one broken business rule: which operator, on which day, and why.
type Violation struct {
	OperatorID int64       `json:"operator_id"`
	Date       pgtype.Date `json:"date"`
	Reason     string      `json:"reason"` // a reason id, <layer>.<entity>.<what>
}

func (v Violation) MarshalProto() *pb.Violation {
	out := &pb.Violation{Date: DateMillis(v.Date), Reason: v.Reason}
	if v.OperatorID != 0 {
		out.Agent = &pb.LookupEntity{Id: v.OperatorID}
	}

	return out
}

type refusalKey string

// Not a plain string, so the error interceptor writes the list as JSON, not as "key = value" text.
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
