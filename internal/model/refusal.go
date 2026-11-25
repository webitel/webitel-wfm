package model

import (
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

// Violation is one broken business rule: which agent, on which day, and why; with the other
// schedule or the existing shift the rule is about, when there is one.
type Violation struct {
	AgentID  int64       `json:"agent_id"`
	Date     pgtype.Date `json:"date"`
	Reason   string      `json:"reason"` // a reason id, <layer>.<entity>.<what>
	Schedule *LookupItem `json:"schedule"`
	ShiftID  int64       `json:"shift_id"`
	StartAt  time.Time   `json:"start_at"`
	EndAt    time.Time   `json:"end_at"`
}

func (v Violation) MarshalProto() *pb.Violation {
	out := &pb.Violation{Date: DateMillis(v.Date), Reason: v.Reason, ShiftId: v.ShiftID}
	if v.AgentID != 0 {
		out.Agent = &pb.LookupEntity{Id: v.AgentID}
	}

	if v.Schedule != nil {
		out.Schedule = v.Schedule.MarshalProto()
	}

	if v.ShiftID != 0 {
		out.StartAt, out.EndAt = v.StartAt.UnixMilli(), v.EndAt.UnixMilli()
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
