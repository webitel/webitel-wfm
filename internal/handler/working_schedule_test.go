package handler

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"
	"github.com/webitel/webitel-go-kit/pkg/etag"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
)

// A schedule state is numbered alike in the database, the model and the
// contract (Core design §1.3), so it keeps its name both ways across the API.
func TestWorkingScheduleStateKeepsItsName(t *testing.T) {
	ctx := grpccontext.SetUser(context.Background(), &model.SignedInUser{ID: 1, DomainID: 2})

	for _, state := range []model.WorkingScheduleState{
		model.WorkingScheduleStateDraft,
		model.WorkingScheduleStatePending,
		model.WorkingScheduleStateActive,
		model.WorkingScheduleStateArchived,
	} {
		name := "WORKING_SCHEDULE_STATE_" + strings.ToUpper(state.String())

		if got := (&model.WorkingSchedule{State: state}).MarshalProto().GetState().String(); got != name {
			t.Errorf("model %s goes out as %s, want %s", state, got, name)
		}

		svc := &updateService{}

		_, err := NewWorkingSchedule(svc).UpdateWorkingScheduleState(ctx, &pb.UpdateWorkingScheduleStateRequest{
			Id: 7, State: pb.WorkingScheduleState(pb.WorkingScheduleState_value[name]), Etag: must(etag.EncodeEtag(etag.EtagWorkingSchedule, 7, 3)),
		})
		if err != nil || svc.got.ID != 7 || svc.got.Ver != 3 || svc.got.State != state {
			t.Errorf("%s comes in as model %+v, %v, want %s", name, svc.got, err, state)
		}
	}
}

// The bounds of a schedule are calendar dates both ways: 00:00 UTC of the day, in ms.
func TestWorkingScheduleDatesAreCalendarDates(t *testing.T) {
	from, to := utcDay(2026, 11, 1), utcDay(2026, 11, 30)

	in := unmarshalWorkingScheduleProto(&pb.WorkingSchedule{StartDateAt: from, EndDateAt: to})
	if in.StartDateAt.Time.Day() != 1 || in.EndDateAt.Time.Day() != 30 {
		t.Errorf("dates come in as %s..%s", in.StartDateAt.Time, in.EndDateAt.Time)
	}

	out := in.MarshalProto()
	if out.GetStartDateAt() != from || out.GetEndDateAt() != to {
		t.Errorf("dates go back out as %d..%d, want %d..%d", out.GetStartDateAt(), out.GetEndDateAt(), from, to)
	}
}

func TestWorkingScheduleEmptyTimeScaleIsTheWholeDay(t *testing.T) {
	if in := unmarshalWorkingScheduleProto(&pb.WorkingSchedule{}); in.StartTimeAt != 0 || in.EndTimeAt != 1440 {
		t.Errorf("an empty scale comes in as %d..%d, want 0..1440", in.StartTimeAt, in.EndTimeAt)
	}

	if in := unmarshalWorkingScheduleProto(&pb.WorkingSchedule{StartTimeAt: 1320, EndTimeAt: 360}); in.StartTimeAt != 1320 || in.EndTimeAt != 360 {
		t.Errorf("a scale past midnight comes in as %d..%d", in.StartTimeAt, in.EndTimeAt)
	}
}

// The etag carries the id and version the row was read at. A row whose mask
// left ver out has none: a token built from 0 would only cause false conflicts.
func TestWorkingScheduleEtagCarriesTheVersion(t *testing.T) {
	if got := (&model.WorkingSchedule{DomainRecord: model.DomainRecord{ID: 7}}).MarshalProto().GetEtag(); got != "" {
		t.Errorf("etag without a selected version = %q, want empty", got)
	}

	tag := (&model.WorkingSchedule{DomainRecord: model.DomainRecord{ID: 7}, Ver: 3}).MarshalProto().GetEtag()

	typ, id, ver, err := etag.DecodeEtag(tag)
	if err != nil || typ != etag.EtagWorkingSchedule || id != 7 || ver != 3 {
		t.Errorf("etag %q decodes to (%s, %d, %d, %v), want (EtagWorkingSchedule, 7, 3, nil)", tag, typ, id, ver, err)
	}
}

// updateService records what reaches the service; anything else panics.
type updateService struct {
	service.WorkingScheduleManager

	got *model.WorkingSchedule
}

func (s *updateService) UpdateWorkingSchedule(_ context.Context, _ *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	s.got = in

	return in, nil
}

func (s *updateService) UpdateWorkingScheduleBlockOutsideActivity(_ context.Context, read *options.Read, ver int32, block bool) (*model.WorkingSchedule, error) {
	s.got = &model.WorkingSchedule{DomainRecord: model.DomainRecord{ID: read.ID()}, Ver: ver, BlockOutsideActivity: block}

	return s.got, nil
}

func (s *updateService) UpdateWorkingScheduleState(_ context.Context, read *options.Read, ver int32, state model.WorkingScheduleState) (*model.WorkingSchedule, error) {
	s.got = &model.WorkingSchedule{DomainRecord: model.DomainRecord{ID: read.ID()}, Ver: ver, State: state}

	return s.got, nil
}

// An update, the block switch and a state change need the etag of the very schedule they name. Without a usable one
// it is a bad request, never the version conflict, and nothing is written.
func TestUpdateWorkingScheduleRequiresItsEtag(t *testing.T) {
	ctx := grpccontext.SetUser(context.Background(), &model.SignedInUser{ID: 1, DomainID: 2})

	own, err := etag.EncodeEtag(etag.EtagWorkingSchedule, 7, 3)
	if err != nil {
		t.Fatal(err)
	}

	for name, tag := range map[string]string{
		"missing":    "",
		"malformed":  "not-an-etag",
		"wrong type": must(etag.EncodeEtag(etag.EtagArticle, 7, 3)),
		"another id": must(etag.EncodeEtag(etag.EtagWorkingSchedule, 8, 3)),
	} {
		t.Run(name, func(t *testing.T) {
			svc := &updateService{}

			_, err := NewWorkingSchedule(svc).UpdateWorkingSchedule(ctx, &pb.UpdateWorkingScheduleRequest{Item: &pb.WorkingSchedule{Id: 7, Etag: tag}})
			if !errors.Is(err, ErrWorkingScheduleEtag) || errors.Code(err) != codes.InvalidArgument {
				t.Errorf("err = %v (%s), want %v", err, errors.Code(err), ErrWorkingScheduleEtag)
			}

			_, err = NewWorkingSchedule(svc).UpdateWorkingScheduleBlockOutsideActivity(ctx, &pb.UpdateWorkingScheduleBlockOutsideActivityRequest{Id: 7, Etag: tag})
			if !errors.Is(err, ErrWorkingScheduleEtag) {
				t.Errorf("block switch: err = %v, want %v", err, ErrWorkingScheduleEtag)
			}

			_, err = NewWorkingSchedule(svc).UpdateWorkingScheduleState(ctx, &pb.UpdateWorkingScheduleStateRequest{Id: 7, State: pb.WorkingScheduleState_WORKING_SCHEDULE_STATE_ACTIVE, Etag: tag})
			if !errors.Is(err, ErrWorkingScheduleEtag) {
				t.Errorf("state: err = %v, want %v", err, ErrWorkingScheduleEtag)
			}

			if svc.got != nil {
				t.Errorf("service was called with %+v", svc.got)
			}
		})
	}

	t.Run("own etag", func(t *testing.T) {
		svc := &updateService{}

		if _, err := NewWorkingSchedule(svc).UpdateWorkingSchedule(ctx, &pb.UpdateWorkingScheduleRequest{Item: &pb.WorkingSchedule{Id: 7, Etag: own}}); err != nil {
			t.Fatal(err)
		}

		if svc.got == nil || svc.got.ID != 7 || svc.got.Ver != 3 {
			t.Errorf("service got %+v, want id 7 at version 3", svc.got)
		}

		svc.got = nil
		if _, err := NewWorkingSchedule(svc).UpdateWorkingScheduleBlockOutsideActivity(ctx, &pb.UpdateWorkingScheduleBlockOutsideActivityRequest{Id: 7, Etag: own, BlockOutsideActivity: true}); err != nil {
			t.Fatal(err)
		}

		if svc.got == nil || svc.got.ID != 7 || svc.got.Ver != 3 || !svc.got.BlockOutsideActivity {
			t.Errorf("block switch got %+v, want id 7 at version 3, switched on", svc.got)
		}
	})
}

func must(tag string, err error) string {
	if err != nil {
		panic(err)
	}

	return tag
}
