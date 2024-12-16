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

type pauseTemplateTestSuite struct {
	suite.Suite

	log *wlog.Logger

	srv *testinfra.TestServer
	cli pb.PauseTemplateServiceClient

	store *pauseTemplates
}

type pauseTemplates struct {
	mu    sync.RWMutex
	items map[int64]*model.PauseTemplate
}

func TestPauseTemplateHandler(t *testing.T) {
	suite.Run(t, new(pauseTemplateTestSuite))
}

func (s *pauseTemplateTestSuite) SetupSuite() {
	s.log = wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleLevel:  wlog.LevelDebug,
	})

	s.srv = testinfra.NewTestServer(s.T(), s.log)
	s.cli = pb.NewPauseTemplateServiceClient(testinfra.NewTestGrpcClient(s.T(), s.srv.Lis))
	s.store = &pauseTemplates{
		items: make(map[int64]*model.PauseTemplate),
	}

	svc := s.mockPauseTemplateServiceBehavior()
	pb.RegisterPauseTemplateServiceServer(s.srv.Server, grpchandler.NewPauseTemplate(svc))

	go func() {
		if err := s.srv.Serve(); err != nil {
			// s.T().Errorf("grpc serve: %v", err)

			return
		}
	}()
}

func (s *pauseTemplateTestSuite) TearDownSuite() {
	if err := s.srv.Lis.Close(); err != nil {
		s.T().Errorf("close grpc listener: %v", err)

		return
	}
}

func (s *pauseTemplateTestSuite) TestCreatePauseTemplate() {
	s.srv.Session.Scopes = []auth_manager.SessionPermission{{
		Name:   "pause_template",
		Access: auth_manager.PERMISSION_ACCESS_CREATE.Value(),
		Obac:   true,
	}}

	type expectation struct {
		out  *pb.CreatePauseTemplateResponse
		code codes.Code
	}

	t := map[string]struct {
		in       *pb.CreatePauseTemplateRequest
		expected expectation
	}{
		"success": {
			in: &pb.CreatePauseTemplateRequest{
				Item: &pb.PauseTemplate{
					Name:        "Test",
					Description: nil,
				},
			},
			expected: expectation{
				out: &pb.CreatePauseTemplateResponse{
					Item: &pb.PauseTemplate{
						Id:          1,
						DomainId:    s.srv.Session.DomainId,
						Name:        "Test",
						Description: nil,
						CreatedBy: &pb.LookupEntity{
							Id: s.srv.Session.UserId,
						},
						UpdatedBy: &pb.LookupEntity{
							Id: s.srv.Session.UserId,
						},
					},
				},
				code: codes.OK,
			},
		},
		"empty name": {
			in: &pb.CreatePauseTemplateRequest{
				Item: &pb.PauseTemplate{
					Name: "",
				},
			},
			expected: expectation{
				code: codes.InvalidArgument,
			},
		},
		"invalid name": {
			in: &pb.CreatePauseTemplateRequest{
				Item: &pb.PauseTemplate{
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
			out, err := s.cli.CreatePauseTemplate(s.srv.Ctx, tt.in)
			if err != nil {
				if e, ok := status.FromError(err); ok {
					s.Require().Equal(tt.expected.code.String(), e.Code().String(), e.Message())
				}
			}

			s.Require().True(cmp.Equal(tt.expected.out, out, cmp.Comparer(proto.Equal)))
		})
	}
}

func (s *pauseTemplateTestSuite) TestReadPauseTemplate() {
	s.srv.Session.Scopes = []auth_manager.SessionPermission{{
		Name:   "pause_template",
		Access: auth_manager.PERMISSION_ACCESS_READ.Value(),
		Obac:   true,
	}}

	type expectation struct {
		out  *pb.ReadPauseTemplateResponse
		code codes.Code
	}

	// TODO: add more test cases
	t := map[string]struct {
		in       *pb.ReadPauseTemplateRequest
		expected expectation
	}{
		"success": {
			in: &pb.ReadPauseTemplateRequest{
				Id: 1,
			},
			expected: expectation{
				out: &pb.ReadPauseTemplateResponse{
					Item: &pb.PauseTemplate{
						Id:          1,
						DomainId:    s.srv.Session.DomainId,
						Name:        "Test",
						Description: nil,
						CreatedBy: &pb.LookupEntity{
							Id: s.srv.Session.UserId,
						},
						UpdatedBy: &pb.LookupEntity{
							Id: s.srv.Session.UserId,
						},
					},
				},
				code: codes.OK,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			out, err := s.cli.ReadPauseTemplate(s.srv.Ctx, tt.in)
			if err != nil {
				if e, ok := status.FromError(err); ok {
					s.Require().Equal(tt.expected.code.String(), e.Code().String(), e.Message())
				}
			}

			s.Require().True(cmp.Equal(tt.expected.out, out, cmp.Comparer(proto.Equal)))
		})
	}
}

func (s *pauseTemplateTestSuite) TestSearchPauseTemplate() {

}

func (s *pauseTemplateTestSuite) TestUpdatePauseTemplate() {
	s.srv.Session.Scopes = []auth_manager.SessionPermission{{
		Name:   "pause_template",
		Access: auth_manager.PERMISSION_ACCESS_UPDATE.Value(),
		Obac:   true,
	}}

	type expectation struct {
		out  *pb.UpdatePauseTemplateResponse
		code codes.Code
	}

	// TODO: add more test cases
	t := map[string]struct {
		in       *pb.UpdatePauseTemplateRequest
		expected expectation
	}{
		"success": {
			in: &pb.UpdatePauseTemplateRequest{
				Item: &pb.PauseTemplate{
					Id:   1,
					Name: "Updated Test",
				},
			},
			expected: expectation{
				out: &pb.UpdatePauseTemplateResponse{
					Item: &pb.PauseTemplate{
						Id:          1,
						DomainId:    s.srv.Session.DomainId,
						Name:        "Updated Test",
						Description: nil,
						CreatedBy: &pb.LookupEntity{
							Id: s.srv.Session.UserId,
						},
						UpdatedBy: &pb.LookupEntity{
							Id: s.srv.Session.UserId,
						},
					},
				},
				code: codes.OK,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			out, err := s.cli.UpdatePauseTemplate(s.srv.Ctx, tt.in)
			if err != nil {
				if e, ok := status.FromError(err); ok {
					s.Require().Equal(tt.expected.code.String(), e.Code().String(), e.Message())
				}
			}

			s.Require().True(cmp.Equal(tt.expected.out, out, cmp.Comparer(proto.Equal)))
		})
	}
}

func (s *pauseTemplateTestSuite) TestDeletePauseTemplate() {
	s.srv.Session.Scopes = []auth_manager.SessionPermission{{
		Name:   "pause_template",
		Access: auth_manager.PERMISSION_ACCESS_DELETE.Value(),
		Obac:   true,
	}}

	type expectation struct {
		out  *pb.DeletePauseTemplateResponse
		code codes.Code
	}

	// TODO: add more test cases
	t := map[string]struct {
		in       *pb.DeletePauseTemplateRequest
		expected expectation
	}{
		"success": {
			in: &pb.DeletePauseTemplateRequest{
				Id: 2,
			},
			expected: expectation{
				out: &pb.DeletePauseTemplateResponse{
					Id: 2,
				},
				code: codes.OK,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			out, err := s.cli.DeletePauseTemplate(s.srv.Ctx, tt.in)
			if err != nil {
				if e, ok := status.FromError(err); ok {
					s.Require().Equal(tt.expected.code.String(), e.Code().String(), e.Message())
				}
			}

			s.Require().True(cmp.Equal(tt.expected.out, out, cmp.Comparer(proto.Equal)))
		})
	}
}

func (s *pauseTemplateTestSuite) TestSearchPauseTemplateCause() {

}

func (s *pauseTemplateTestSuite) TestUpdatePauseTemplateCauseBulk() {

}

func (s *pauseTemplateTestSuite) mockPauseTemplateServiceBehavior() *handler.MockPauseTemplateManager {
	s.T().Helper()
	svc := handler.NewMockPauseTemplateManager(s.T())
	svc.EXPECT().CreatePauseTemplate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) (int64, error) {
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

	svc.EXPECT().ReadPauseTemplate(mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.PauseTemplate, error) {
			s.store.mu.RLock()
			te, ok := s.store.items[id]
			s.store.mu.RUnlock()
			if !ok {
				return nil, werror.NewDBNoRowsErr("tests")
			}

			return te, nil
		},
	).Maybe()

	svc.EXPECT().SearchPauseTemplate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.PauseTemplate, bool, error) {
			out := make([]*model.PauseTemplate, 0, len(s.store.items))
			for i, v := range s.store.items {
				v.Id = i
				out = append(out, v)
			}

			return out, false, nil
		},
	).Maybe()

	svc.EXPECT().UpdatePauseTemplate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) error {
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

	svc.EXPECT().DeletePauseTemplate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
			s.store.mu.Lock()
			delete(s.store.items, id)
			s.store.mu.Unlock()

			return id, nil
		},
	).Maybe()

	return svc
}
