package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	servicemock "github.com/webitel/webitel-wfm/gen/go/mocks/service"
	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/internal/handler"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/service"
	"github.com/webitel/webitel-wfm/internal/tests"
	"github.com/webitel/webitel-wfm/pkg/werror/old"
)

type pauseTemplateTestSuite struct {
	suite.Suite

	log *wlog.Logger
	svc handler.PauseTemplateManager
	c   cache.Manager

	store *pauseTemplates
}

type pauseTemplates struct {
	mu    sync.RWMutex
	items map[int64]*model.PauseTemplate
}

func TestPauseTemplateService(t *testing.T) {
	suite.Run(t, new(pauseTemplateTestSuite))
}

func (s *pauseTemplateTestSuite) SetupSuite() {
	s.log = wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleLevel:  wlog.LevelDebug,
	})

	s.svc = service.NewPauseTemplate(s.mockPauseTemplateRepositoryBehavior())
	s.store = &pauseTemplates{
		items: make(map[int64]*model.PauseTemplate),
	}
}

func (s *pauseTemplateTestSuite) TearDownSuite() {}

func (s *pauseTemplateTestSuite) TestCreatePauseTemplate() {
	ctx := context.Background()
	user := tests.User(auth_manager.PERMISSION_ACCESS_CREATE.Value(), false)
	type expectation struct {
		out int64
	}

	t := map[string]struct {
		user     *model.SignedInUser
		in       *model.PauseTemplate
		expected expectation
	}{
		"success": {
			user: user,
			in: &model.PauseTemplate{
				DomainRecord: model.DomainRecord{
					Id:       1,
					DomainId: 1,
					CreatedBy: model.LookupItem{
						Id: 1,
					},
					UpdatedBy: model.LookupItem{
						Id: 1,
					},
				},
				Name:        "foo",
				Description: tests.ValueToPTR[string]("bar"),
			},
			expected: expectation{
				out: 1,
			},
		},
		"nil description": {
			user: user,
			in: &model.PauseTemplate{
				DomainRecord: model.DomainRecord{
					Id:       2,
					DomainId: 1,
					CreatedBy: model.LookupItem{
						Id: 1,
					},
					UpdatedBy: model.LookupItem{
						Id: 1,
					},
				},
				Name:        "foo",
				Description: nil,
			},
			expected: expectation{
				out: 2,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			out, err := s.svc.CreatePauseTemplate(ctx, tt.user, tt.in)
			s.Require().Nil(err)
			s.Require().Equal(tt.expected.out, out)
		})
	}
}

func (s *pauseTemplateTestSuite) TestReadPauseTemplate() {
	ctx := context.Background()
	user := tests.User(auth_manager.PERMISSION_ACCESS_READ.Value(), false)
	type expectation struct {
		out    *model.PauseTemplate
		cached bool
	}

	t := map[string]struct {
		user     *model.SignedInUser
		in       *model.SearchItem
		expected expectation
	}{
		"success": {
			user: user,
			in: &model.SearchItem{
				Id: 1,
			},
			expected: expectation{
				out: &model.PauseTemplate{
					DomainRecord: model.DomainRecord{
						Id:       1,
						DomainId: 1,
						CreatedBy: model.LookupItem{
							Id: 1,
						},
						UpdatedBy: model.LookupItem{
							Id: 1,
						},
					},
					Name:        "foo",
					Description: tests.ValueToPTR[string]("bar"),
				},
				cached: false,
			},
		},
		"success with fields": {
			user: user,
			in: &model.SearchItem{
				Id:     1,
				Fields: []string{"id", "name"},
			},
			expected: expectation{
				out: &model.PauseTemplate{
					DomainRecord: model.DomainRecord{
						Id:       1,
						DomainId: 1,
						CreatedBy: model.LookupItem{
							Id: 1,
						},
						UpdatedBy: model.LookupItem{
							Id: 1,
						},
					},
					Name:        "foo",
					Description: tests.ValueToPTR[string]("bar"),
				},
				cached: false,
			},
		},
		"success with other fields": {
			user: user,
			in: &model.SearchItem{
				Id:     1,
				Fields: []string{"id", "name", "description"},
			},
			expected: expectation{
				out: &model.PauseTemplate{
					DomainRecord: model.DomainRecord{
						Id:       1,
						DomainId: 1,
						CreatedBy: model.LookupItem{
							Id: 1,
						},
						UpdatedBy: model.LookupItem{
							Id: 1,
						},
					},
					Name:        "foo",
					Description: tests.ValueToPTR[string]("bar"),
				},
				cached: false,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			out, err := s.svc.ReadPauseTemplate(ctx, tt.user, tt.in)
			s.Require().Nil(err)
			s.Require().Equal(tt.expected.out, out)
		})
	}
}

func (s *pauseTemplateTestSuite) TestSearchPauseTemplate() {}

func (s *pauseTemplateTestSuite) TestUpdatePauseTemplate() {}

func (s *pauseTemplateTestSuite) TestDeletePauseTemplate() {}

func (s *pauseTemplateTestSuite) mockPauseTemplateRepositoryBehavior() *servicemock.MockPauseTemplateManager {
	s.T().Helper()
	r := servicemock.NewMockPauseTemplateManager(s.T())
	r.EXPECT().CreatePauseTemplate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) (int64, error) {
			s.store.mu.Lock()
			s.store.items[in.Id] = in
			s.store.mu.Unlock()

			return in.Id, nil
		},
	).Maybe()

	r.EXPECT().SearchPauseTemplate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, in *model.SearchItem) ([]*model.PauseTemplate, error) {
			out := make([]*model.PauseTemplate, 0, len(s.store.items))
			if in.Id != 0 {
				v := s.store.items[in.Id]
				out = append(out, v)

				return out, nil
			}

			for _, v := range s.store.items {
				out = append(out, v)
			}

			return out, nil
		},
	).Maybe()

	r.EXPECT().UpdatePauseTemplate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) error {
			s.store.mu.RLock()
			_, ok := s.store.items[in.Id]
			s.store.mu.RUnlock()
			if !ok {
				return werror.NewDBNoRowsErr("tests")
			}

			s.store.mu.Lock()
			s.store.items[in.Id] = in
			s.store.mu.Unlock()

			return nil
		},
	).Maybe()

	r.EXPECT().DeletePauseTemplate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
			s.store.mu.Lock()
			delete(s.store.items, id)
			s.store.mu.Unlock()

			return id, nil
		},
	).Maybe()

	return r
}
