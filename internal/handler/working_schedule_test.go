package handler_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/suite"
	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	mockservice "github.com/webitel/webitel-wfm/gen/go/mocks/internal_/service"
	"github.com/webitel/webitel-wfm/internal/handler"
	"github.com/webitel/webitel-wfm/internal/tests/testinfra"
)

type workingScheduleTestSuite struct {
	suite.Suite

	log *wlog.Logger

	srv *testinfra.TestServer
	cli pb.WorkingScheduleServiceClient
}

func TestWorkingScheduleHandler(t *testing.T) {
	suite.Run(t, new(workingScheduleTestSuite))
}

func (s *workingScheduleTestSuite) SetupSuite() {
	s.log = wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleLevel:  wlog.LevelDebug,
	})

	s.srv = testinfra.NewTestServer(s.T(), s.log)
	s.cli = pb.NewWorkingScheduleServiceClient(testinfra.NewTestGrpcClient(s.T(), s.srv.Lis))

	svc := mockservice.NewMockWorkingScheduleManager(s.T())
	_ = handler.NewWorkingSchedule(s.srv.Server, svc)

	go func() {
		if err := s.srv.Serve(); err != nil {
			// s.T().Errorf("grpc serve: %v", err)

			return
		}
	}()
}

func (s *workingScheduleTestSuite) TearDownSuite() {
	if err := s.srv.Lis.Close(); err != nil {
		s.T().Errorf("close grpc listener: %v", err)

		return
	}
}

func (s *workingScheduleTestSuite) TestCreateWorkingCondition() {
	s.srv.Session.Scopes = []auth_manager.SessionPermission{{
		Name:   "working_condition",
		Access: auth_manager.PERMISSION_ACCESS_CREATE.Value(),
		Obac:   true,
	}}

	type expectation struct {
		out  *pb.CreateWorkingScheduleResponse
		code codes.Code
	}

	t := map[string]struct {
		in       *pb.CreateWorkingScheduleRequest
		expected expectation
	}{
		"success": {
			in: &pb.CreateWorkingScheduleRequest{},
			expected: expectation{
				out:  &pb.CreateWorkingScheduleResponse{},
				code: codes.OK,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			out, err := s.cli.CreateWorkingSchedule(s.srv.Ctx, tt.in)
			if err != nil {
				if e, ok := status.FromError(err); ok {
					s.Require().Equal(tt.expected.code.String(), e.Code().String(), e.Message())
				}
			}

			s.Require().True(cmp.Equal(tt.expected.out, out, cmp.Comparer(proto.Equal)))
		})
	}
}

func (s *workingScheduleTestSuite) TestReadWorkingSchedule() {}

func (s *workingScheduleTestSuite) TestSearchWorkingSchedule() {}

func (s *workingScheduleTestSuite) TestUpdateWorkingSchedule() {}

func (s *workingScheduleTestSuite) TestDeleteWorkingSchedule() {}
