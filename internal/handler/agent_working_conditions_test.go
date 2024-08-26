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

	pb "github.com/webitel/webitel-wfm/gen/go/api"
	"github.com/webitel/webitel-wfm/gen/go/mocks/handler"
	grpchandler "github.com/webitel/webitel-wfm/internal/handler"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/tests"
	"github.com/webitel/webitel-wfm/internal/tests/testinfra"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type agentWorkingConditionsTestSuite struct {
	suite.Suite

	log *wlog.Logger

	srv *testinfra.TestServer
	cli pb.AgentWorkingConditionsServiceClient

	store *agentWorkingConditions
}

type agentWorkingConditions struct {
	mu    sync.RWMutex
	items map[int64]*model.AgentWorkingConditions
}

func TestAgentHandler(t *testing.T) {
	suite.Run(t, new(agentWorkingConditionsTestSuite))
}

func (s *agentWorkingConditionsTestSuite) SetupSuite() {
	s.log = wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleLevel:  wlog.LevelDebug,
	})

	s.srv = testinfra.NewTestServer(s.T(), s.log)
	s.cli = pb.NewAgentWorkingConditionsServiceClient(testinfra.NewTestGrpcClient(s.T(), s.srv.Lis))
	s.store = &agentWorkingConditions{
		items: make(map[int64]*model.AgentWorkingConditions),
	}

	s.store.items[1] = &model.AgentWorkingConditions{
		WorkingCondition: model.LookupItem{
			Id:   1,
			Name: tests.ValueToPTR[string]("foo"),
		},
		PauseTemplate: &model.LookupItem{
			Id:   1,
			Name: tests.ValueToPTR[string]("foo"),
		},
	}

	svc := s.mockWorkingConditionsServiceBehavior()
	pb.RegisterAgentWorkingConditionsServiceServer(s.srv.Server, grpchandler.NewAgentWorkingConditions(svc))

	go func() {
		if err := s.srv.Serve(); err != nil {
			// s.T().Errorf("grpc serve: %v", err)

			return
		}
	}()
}

func (s *agentWorkingConditionsTestSuite) TearDownSuite() {
	if err := s.srv.Lis.Close(); err != nil {
		s.T().Errorf("close grpc listener: %v", err)

		return
	}
}

func (s *agentWorkingConditionsTestSuite) TestReadAgentWorkingConditions() {
	s.srv.Session.Scopes = []auth_manager.SessionPermission{{
		Name:   "agent",
		Access: auth_manager.PERMISSION_ACCESS_READ.Value(),
		Obac:   true,
	}}

	type expectation struct {
		out  *pb.ReadAgentWorkingConditionsResponse
		code codes.Code
	}

	// TODO: add more test cases
	t := map[string]struct {
		in       *pb.ReadAgentWorkingConditionsRequest
		expected expectation
	}{
		"success": {
			in: &pb.ReadAgentWorkingConditionsRequest{
				AgentId: 1,
			},
			expected: expectation{
				out: &pb.ReadAgentWorkingConditionsResponse{
					Item: &pb.AgentWorkingConditions{
						WorkingCondition: &pb.LookupEntity{
							Id:   1,
							Name: tests.ValueToPTR[string]("foo"),
						},
						PauseTemplate: &pb.LookupEntity{
							Id:   1,
							Name: tests.ValueToPTR[string]("foo"),
						},
					},
				},
				code: codes.OK,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			out, err := s.cli.ReadAgentWorkingConditions(s.srv.Ctx, tt.in)
			if err != nil {
				if e, ok := status.FromError(err); ok {
					s.Require().Equal(tt.expected.code.String(), e.Code().String(), e.Message())
				}
			}

			s.Require().True(cmp.Equal(tt.expected.out, out, cmp.Comparer(proto.Equal)))
		})
	}
}

func (s *agentWorkingConditionsTestSuite) TestUpdateAgentWorkingConditions() {
	s.srv.Session.Scopes = []auth_manager.SessionPermission{{
		Name:   "agent",
		Access: auth_manager.PERMISSION_ACCESS_UPDATE.Value(),
		Obac:   true,
	}}

	type expectation struct {
		out  *pb.UpdateAgentWorkingConditionsResponse
		code codes.Code
	}

	// TODO: add more test cases
	t := map[string]struct {
		in       *pb.UpdateAgentWorkingConditionsRequest
		expected expectation
	}{
		"success": {
			in: &pb.UpdateAgentWorkingConditionsRequest{
				AgentId: 1,
				Item: &pb.AgentWorkingConditions{
					WorkingCondition: &pb.LookupEntity{
						Id: 2,
					},
					PauseTemplate: &pb.LookupEntity{
						Id: 2,
					},
				},
			},
			expected: expectation{
				out: &pb.UpdateAgentWorkingConditionsResponse{
					Item: &pb.AgentWorkingConditions{
						WorkingCondition: &pb.LookupEntity{
							Id: 2,
						},
						PauseTemplate: &pb.LookupEntity{
							Id: 2,
						},
					},
				},
				code: codes.OK,
			},
		},
		"without pause template": {
			in: &pb.UpdateAgentWorkingConditionsRequest{
				AgentId: 1,
				Item: &pb.AgentWorkingConditions{
					WorkingCondition: &pb.LookupEntity{
						Id: 2,
					},
				},
			},
			expected: expectation{
				out: &pb.UpdateAgentWorkingConditionsResponse{
					Item: &pb.AgentWorkingConditions{
						WorkingCondition: &pb.LookupEntity{
							Id: 2,
						},
						PauseTemplate: nil,
					},
				},
				code: codes.OK,
			},
		},
		"without working condition": {
			in: &pb.UpdateAgentWorkingConditionsRequest{
				AgentId: 1,
				Item: &pb.AgentWorkingConditions{
					PauseTemplate: &pb.LookupEntity{
						Id: 2,
					},
				},
			},
			expected: expectation{
				out:  nil,
				code: codes.InvalidArgument,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			out, err := s.cli.UpdateAgentWorkingConditions(s.srv.Ctx, tt.in)
			if err != nil {
				if e, ok := status.FromError(err); ok {
					s.Require().Equal(tt.expected.code.String(), e.Code().String(), e.Message())
				}
			}

			s.Require().True(cmp.Equal(tt.expected.out, out, cmp.Comparer(proto.Equal)))
		})
	}
}

func (s *agentWorkingConditionsTestSuite) mockWorkingConditionsServiceBehavior() *handler.MockAgentWorkingConditionsManager {
	s.T().Helper()
	svc := handler.NewMockAgentWorkingConditionsManager(s.T())
	svc.EXPECT().ReadAgentWorkingConditions(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, user *model.SignedInUser, agentId int64) (*model.AgentWorkingConditions, error) {
			s.store.mu.RLock()
			te, ok := s.store.items[agentId]
			s.store.mu.RUnlock()
			if !ok {
				return nil, werror.NewDBNoRowsErr("tests")
			}

			return te, nil
		}).Maybe()

	svc.EXPECT().UpdateAgentWorkingConditions(mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, agentId int64, in *model.AgentWorkingConditions) error {
			s.store.mu.RLock()
			_, ok := s.store.items[agentId]
			s.store.mu.RUnlock()
			if !ok {
				return werror.NewDBNoRowsErr("tests")
			}

			s.store.mu.Lock()
			s.store.items[agentId] = in
			s.store.mu.Unlock()

			return nil
		},
	).Maybe()

	return svc
}
