package handler

import (
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/internal/model"
)

// A range of absence covers both of its calendar dates, for every agent of the request.
func TestAgentsAbsencesCoverTheRange(t *testing.T) {
	out, err := unmarshalAgentsAbsencesBulk([]int64{1, 2}, []*pb.CreateAgentsAbsencesRequestAbsentType{
		{TypeId: pb.AbsenceType_ABSENCE_TYPE_VACATION, DateFrom: "2026-11-01", DateTo: "2026-11-03"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(out) != 2 {
		t.Fatalf("agents = %d, want 2", len(out))
	}

	days := make([]string, 0, len(out[0].Absence))
	for _, a := range out[0].Absence {
		days = append(days, model.FormatDate(a.AbsentAt))
	}

	if len(days) != 3 || days[0] != "2026-11-01" || days[2] != "2026-11-03" {
		t.Errorf("days = %v, want 2026-11-01..2026-11-03", days)
	}

	if _, err := unmarshalAgentsAbsencesBulk([]int64{1}, []*pb.CreateAgentsAbsencesRequestAbsentType{{DateFrom: "2026-02-30", DateTo: "2026-03-01"}}); !errors.Is(err, ErrDate) {
		t.Errorf("err = %v, want ErrDate", err)
	}
}

func TestAbsenceDateBothWays(t *testing.T) {
	in, err := unmarshalAbsenceProto(&pb.Absence{AbsentAt: "2026-11-04", TypeId: pb.AbsenceType_ABSENCE_TYPE_DAYOFF})
	if err != nil {
		t.Fatal(err)
	}

	if got := in.MarshalProto().GetAbsentAt(); got != "2026-11-04" {
		t.Errorf("absent_at goes back out as %q", got)
	}
}
