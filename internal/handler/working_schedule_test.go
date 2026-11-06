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
	"github.com/webitel/webitel-wfm/internal/service"
)

// A schedule state is numbered alike in the database, the model and the
// contract (Core design §1.3), so it keeps its name both ways across the API.
func TestWorkingScheduleStateKeepsItsName(t *testing.T) {
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

		in := unmarshalWorkingScheduleProto(&pb.WorkingSchedule{
			State:    pb.WorkingScheduleState(pb.WorkingScheduleState_value[name]),
			Team:     &pb.LookupEntity{},
			Calendar: &pb.LookupEntity{},
		})
		if in.State != state {
			t.Errorf("%s comes in as model %s, want %s", name, in.State, state)
		}
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

// An update needs the etag of the very schedule it names. Without a usable one
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
	})
}

func must(tag string, err error) string {
	if err != nil {
		panic(err)
	}

	return tag
}
