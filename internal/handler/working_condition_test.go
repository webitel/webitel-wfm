package handler

import (
	"testing"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

// Every number of the working conditions may be left out; days off and pauses may be 0, a workday may not.
func TestValidatorTakesLookupBounds(t *testing.T) {
	v, err := protovalidate.New()
	if err != nil {
		t.Fatal(err)
	}

	wc := func(c *pb.WorkingCondition) *pb.WorkingCondition {
		c.Name = "conditions"

		return c
	}
	pauses := func(minutes int64) *pb.PauseTemplate {
		return &pb.PauseTemplate{Name: "pauses", Causes: []*pb.PauseTemplateCause{{Duration: &minutes}}}
	}

	for name, tc := range map[string]struct {
		msg proto.Message
		ok  bool
	}{
		"all left out":                {msg: wc(&pb.WorkingCondition{}), ok: true},
		"no days off and no pause":    {msg: wc(&pb.WorkingCondition{Vacation: ptr(int32(0)), SickLeaves: ptr(int32(0)), DaysOff: ptr(int32(0)), PauseDuration: ptr(int32(0))}), ok: true},
		"a workday of 0 hours":        {msg: wc(&pb.WorkingCondition{WorkdayHours: ptr(int32(0))})},
		"0 workdays a month":          {msg: wc(&pb.WorkingCondition{WorkdaysPerMonth: ptr(int32(0))})},
		"a pause without a workday":   {msg: wc(&pb.WorkingCondition{PauseDuration: ptr(int32(90))}), ok: true},
		"a pause as long as the day":  {msg: wc(&pb.WorkingCondition{WorkdayHours: ptr(int32(1)), PauseDuration: ptr(int32(60))}), ok: true},
		"a pause a minute longer":     {msg: wc(&pb.WorkingCondition{WorkdayHours: ptr(int32(1)), PauseDuration: ptr(int32(61))})},
		"the whole year off":          {msg: wc(&pb.WorkingCondition{Vacation: ptr(int32(200)), SickLeaves: ptr(int32(100)), DaysOff: ptr(int32(65))}), ok: true},
		"more than a year off":        {msg: wc(&pb.WorkingCondition{Vacation: ptr(int32(200)), SickLeaves: ptr(int32(100)), DaysOff: ptr(int32(66))})},
		"a template pause of 0":       {msg: pauses(0), ok: true},
		"a template pause of a day":   {msg: pauses(1440), ok: true},
		"a template pause over a day": {msg: pauses(1441)},
	} {
		if err := v.Validate(tc.msg); (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, want valid %v", name, err, tc.ok)
		}
	}
}
