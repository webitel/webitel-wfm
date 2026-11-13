package handler

import (
	"testing"
	"time"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/internal/model"
)

// A range of absence covers both of its dates, for every agent of the request.
func TestAgentsAbsencesCoverTheRange(t *testing.T) {
	out := unmarshalAgentsAbsencesBulk([]int64{1, 2}, []*pb.CreateAgentsAbsencesRequestAbsentType{
		{TypeId: pb.AbsenceType_ABSENCE_TYPE_VACATION, DateFrom: utcDay(2026, 11, 1), DateTo: utcDay(2026, 11, 3)},
	})
	if len(out) != 2 {
		t.Fatalf("agents = %d, want 2", len(out))
	}

	days := make([]int64, 0, len(out[0].Absence))
	for _, a := range out[0].Absence {
		days = append(days, model.DateMillis(a.AbsentAt))
	}

	if len(days) != 3 || days[0] != utcDay(2026, 11, 1) || days[2] != utcDay(2026, 11, 3) {
		t.Errorf("days = %v, want 2026-11-01..2026-11-03 as UTC midnights", days)
	}
}

func TestAbsenceDateBothWays(t *testing.T) {
	in := unmarshalAbsenceProto(&pb.Absence{AbsentAt: utcDay(2026, 11, 4), TypeId: pb.AbsenceType_ABSENCE_TYPE_DAYOFF})
	if got := in.MarshalProto().GetAbsentAt(); got != utcDay(2026, 11, 4) {
		t.Errorf("absent_at goes back out as %d", got)
	}
}

// A date is 00:00 UTC of its day. A client's local midnight, which a date picker gives by
// default, is refused by the validator instead of landing on the day before in UTC+ zones.
func TestValidatorTakesDatesAsUTCMidnights(t *testing.T) {
	v, err := protovalidate.New()
	if err != nil {
		t.Fatal(err)
	}

	kyiv, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		t.Fatal(err)
	}

	local := time.Date(2026, 11, 20, 0, 0, 0, 0, kyiv).UnixMilli()
	utc := utcDay(2026, 11, 20)

	for name, tc := range map[string]struct {
		msg proto.Message
		ok  bool
	}{
		"absence, UTC midnight":     {msg: &pb.Absence{TypeId: pb.AbsenceType_ABSENCE_TYPE_DAYOFF, AbsentAt: utc}, ok: true},
		"absence, local midnight":   {msg: &pb.Absence{TypeId: pb.AbsenceType_ABSENCE_TYPE_DAYOFF, AbsentAt: local}},
		"range, UTC midnights":      {msg: &pb.DateRange{From: utc, To: utc}, ok: true},
		"range, local midnight":     {msg: &pb.DateRange{From: local, To: utc}},
		"range, end of the day":     {msg: &pb.DateRange{From: utc, To: utc + 86399999}},
		"range, backwards":          {msg: &pb.DateRange{From: utc, To: utcDay(2026, 11, 19)}},
		"bulk absence, UTC":         {msg: &pb.CreateAgentsAbsencesRequestAbsentType{TypeId: pb.AbsenceType_ABSENCE_TYPE_DAYOFF, DateFrom: utc, DateTo: utc}, ok: true},
		"bulk absence, local":       {msg: &pb.CreateAgentsAbsencesRequestAbsentType{TypeId: pb.AbsenceType_ABSENCE_TYPE_DAYOFF, DateFrom: local, DateTo: utc}},
		"schedule bounds, UTC":      {msg: &pb.WorkingSchedule{Name: "s", StartDateAt: utc, EndDateAt: utc}, ok: true},
		"schedule bounds, local":    {msg: &pb.WorkingSchedule{Name: "s", StartDateAt: local}},
		"schedule bounds, negative": {msg: &pb.WorkingSchedule{Name: "s", StartDateAt: -86400000}},
		"schedule bounds left out":  {msg: &pb.WorkingSchedule{Name: "s"}, ok: true},
	} {
		if err := v.Validate(tc.msg); (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, want valid %v", name, err, tc.ok)
		}
	}
}
