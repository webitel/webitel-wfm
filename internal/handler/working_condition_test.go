package handler_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/gen/go/mocks/handler"
	grpchandler "github.com/webitel/webitel-wfm/internal/handler"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/tests"
	"github.com/webitel/webitel-wfm/internal/tests/testinfra"
	"github.com/webitel/webitel-wfm/pkg/werror/old"
)

type workingConditionTestSuite struct {
	suite.Suite

	log *wlog.Logger

	srv *testinfra.TestServer
	cli pb.WorkingConditionServiceClient

	store *workingConditions
}

type workingConditions struct {
	mu    sync.RWMutex
	items map[int64]*model.WorkingCondition
}

func TestWorkingConditionHandler(t *testing.T) {
	suite.Run(t, new(workingConditionTestSuite))
}

func (s *workingConditionTestSuite) SetupSuite() {
	s.log = wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleLevel:  wlog.LevelDebug,
	})

	s.srv = testinfra.NewTestServer(s.T(), s.log)
	s.cli = pb.NewWorkingConditionServiceClient(testinfra.NewTestGrpcClient(s.T(), s.srv.Lis))
	s.store = &workingConditions{
		items: make(map[int64]*model.WorkingCondition),
	}

	svc := s.mockWorkingConditionServiceBehavior()
	pb.RegisterWorkingConditionServiceServer(s.srv.Server, grpchandler.NewWorkingCondition(svc))

	go func() {
		if err := s.srv.Serve(); err != nil {
			// s.T().Errorf("grpc serve: %v", err)

			return
		}
	}()
}

func (s *workingConditionTestSuite) TearDownSuite() {
	if err := s.srv.Lis.Close(); err != nil {
		s.T().Errorf("close grpc listener: %v", err)

		return
	}
}

func (s *workingConditionTestSuite) TestCreateWorkingCondition() {
	s.srv.Session.Scopes = []auth_manager.SessionPermission{{
		Name:   "working_condition",
		Access: auth_manager.PERMISSION_ACCESS_CREATE.Value(),
		Obac:   true,
	}}

	type expectation struct {
		out  *pb.CreateWorkingConditionResponse
		code codes.Code
	}

	t := map[string]struct {
		in       *pb.CreateWorkingConditionRequest
		expected expectation
	}{
		"success": {
			in: &pb.CreateWorkingConditionRequest{
				Item: &pb.WorkingCondition{
					Name:        "Test",
					Description: nil,
					PauseTemplate: &pb.LookupEntity{
						Id: 1,
					},
				},
			},
			expected: expectation{
				out: &pb.CreateWorkingConditionResponse{
					Item: &pb.WorkingCondition{
						Id:       1,
						DomainId: s.srv.Session.DomainId,
						CreatedBy: &pb.LookupEntity{
							Id: s.srv.Session.UserId,
						},
						UpdatedBy: &pb.LookupEntity{
							Id: s.srv.Session.UserId,
						},
						Name:        "Test",
						Description: nil,
						PauseTemplate: &pb.LookupEntity{
							Id: 1,
						},
					},
				},
				code: codes.OK,
			},
		},
		"empty name": {
			in: &pb.CreateWorkingConditionRequest{
				Item: &pb.WorkingCondition{
					Name: "",
				},
			},
			expected: expectation{
				code: codes.InvalidArgument,
			},
		},
		"invalid name": {
			in: &pb.CreateWorkingConditionRequest{
				Item: &pb.WorkingCondition{
					Name: tests.RandStringBytes(300),
				},
			},
			expected: expectation{
				code: codes.InvalidArgument,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			out, err := s.cli.CreateWorkingCondition(s.srv.Ctx, tt.in)
			if err != nil {
				if e, ok := status.FromError(err); ok {
					s.Require().Equal(tt.expected.code.String(), e.Code().String(), e.Message())
				}
			}

			s.Require().True(cmp.Equal(tt.expected.out, out, cmp.Comparer(proto.Equal)))
		})
	}
}

func (s *workingConditionTestSuite) TestReadWorkingCondition() {}

func (s *workingConditionTestSuite) TestSearchWorkingCondition() {}

func (s *workingConditionTestSuite) TestUpdateWorkingCondition() {}

func (s *workingConditionTestSuite) TestDeleteWorkingCondition() {}

func (s *workingConditionTestSuite) mockWorkingConditionServiceBehavior() *handler.MockWorkingConditionManager {
	s.T().Helper()

	svc := handler.NewMockWorkingConditionManager(s.T())
	svc.EXPECT().CreateWorkingCondition(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) (int64, error) {
			in.Id = int64(len(s.store.items) + 1)
			in.DomainId = user.DomainId
			in.CreatedBy = model.LookupItem{Id: user.Id}
			in.UpdatedBy = model.LookupItem{Id: user.Id}

			s.store.mu.Lock()
			s.store.items[in.Id] = in
			s.store.mu.Unlock()

			return in.Id, nil
		},
	).Maybe()

	svc.EXPECT().ReadWorkingCondition(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingCondition, error) {
			s.store.mu.RLock()
			te, ok := s.store.items[search.Id]
			s.store.mu.RUnlock()
			if !ok {
				return nil, werror.NewDBNoRowsErr("tests")
			}

			return te, nil
		},
	).Maybe()

	svc.EXPECT().SearchWorkingCondition(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingCondition, bool, error) {
			out := make([]*model.WorkingCondition, 0, len(s.store.items))
			for i, v := range s.store.items {
				v.Id = i
				out = append(out, v)
			}

			return out, false, nil
		},
	).Maybe()

	svc.EXPECT().UpdateWorkingCondition(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) error {
			s.store.mu.RLock()
			_, ok := s.store.items[in.Id]
			s.store.mu.RUnlock()
			if !ok {
				return werror.NewDBNoRowsErr("tests")
			}

			in.DomainId = user.DomainId
			in.CreatedBy = model.LookupItem{Id: user.Id}
			in.UpdatedBy = model.LookupItem{Id: user.Id}

			s.store.mu.Lock()
			s.store.items[in.Id] = in
			s.store.mu.Unlock()

			return nil
		},
	).Maybe()

	svc.EXPECT().DeleteWorkingCondition(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
			s.store.mu.Lock()
			delete(s.store.items, id)
			s.store.mu.Unlock()

			return id, nil
		},
	).Maybe()

	return svc
}
