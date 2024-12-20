package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/suite"
	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/storage"
	"github.com/webitel/webitel-wfm/internal/tests"
	"github.com/webitel/webitel-wfm/internal/tests/testinfra"
)

type workingScheduleStorage interface {
	CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	ReadWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingSchedule, error)
	SearchWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingSchedule, error)
	UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	DeleteWorkingSchedule(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}

type workingScheduleTestSuite struct {
	suite.Suite

	log     *wlog.Logger
	cluster *testinfra.TestStorageCluster
	cache   cache.Manager

	store workingScheduleStorage
}

func TestNewWorkingSchedule(t *testing.T) {
	suite.Run(t, new(workingScheduleTestSuite))
}

func (s *workingScheduleTestSuite) SetupSuite() {
	var err error

	s.cluster, err = testinfra.NewTestStorageCluster(s.T(), wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: true}))
	if err != nil {
		s.T().Error(err)
	}

	s.cache, err = cache.New(&config.Cache{Size: 1024})
	if err != nil {
		s.T().Error(err)
	}

	s.store = storage.NewWorkingSchedule(s.cluster.Store(), s.cache)
}

func (s *workingScheduleTestSuite) TearDownSuite() {

}

func (s *workingScheduleTestSuite) TestCreateWorkingSchedule() {
	ctx := context.Background()
	user := tests.User(auth_manager.PERMISSION_ACCESS_CREATE.Value(), false)
	type expectation struct {
		out int64
		err error
	}

	// TODO: add more test cases
	t := map[string]struct {
		user     *model.SignedInUser
		in       *model.WorkingSchedule
		expected expectation
	}{
		"success": {
			user: user,
			in: &model.WorkingSchedule{
				DomainRecord:         model.DomainRecord{DomainId: 1, Id: 1},
				Name:                 "foo",
				Team:                 model.LookupItem{Id: 1},
				Calendar:             model.LookupItem{Id: 1},
				StartDateAt:          model.NewDate(time.Now().Unix()),
				EndDateAt:            model.NewDate(time.Now().Unix()),
				StartTimeAt:          0,
				EndTimeAt:            1440,
				ExtraSkills:          []*model.LookupItem{{Id: 1}},
				BlockOutsideActivity: false,
				Agents:               []*model.LookupItem{{Id: 1}},
			},
			expected: expectation{
				out: 1,
				err: nil,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			create := struct {
				sql  string
				args []any
			}{
				sql: "WITH schedule AS (INSERT INTO wfm.working_schedule (block_outside_activity, calendar_id, created_by, " +
					"domain_id, end_date_at, end_time_at, name, start_date_at, start_time_at, state, team_id, updated_by) " +
					"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id), " +
					"extra_skills AS (INSERT INTO wfm.working_schedule_extra_skill (domain_id, skill_id, working_schedule_id) " +
					"VALUES ($13, $14, (SELECT id FROM schedule)::bigint) RETURNING id), " +
					"agents AS (INSERT INTO wfm.working_schedule_agent (agent_id, domain_id, working_schedule_id) " +
					"VALUES ($15, $16, (SELECT id FROM schedule)::bigint) RETURNING id) " +
					"SELECT DISTINCT schedule.id FROM schedule, extra_skills, agents",
				args: []any{
					tt.in.BlockOutsideActivity,
					tt.in.Calendar.SafeId(),
					tt.user.Id,
					tt.user.DomainId,
					tt.in.EndDateAt,
					tt.in.EndTimeAt,
					tt.in.Name,
					tt.in.StartDateAt,
					tt.in.StartTimeAt,
					tt.in.State,
					tt.in.Team.SafeId(),
					tt.user.Id,
				},
			}

			for _, skill := range tt.in.ExtraSkills {
				create.args = append(create.args, tt.user.DomainId, skill.SafeId())
			}

			for _, agent := range tt.in.Agents {
				create.args = append(create.args, agent.SafeId(), tt.user.DomainId)
			}

			rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(1))
			s.cluster.Mock().ExpectQuery(create.sql).WithArgs(create.args...).WillReturnRows(rows)
			out, err := s.store.CreateWorkingSchedule(ctx, tt.user, tt.in)
			if err != nil {
				s.T().Error(err)
			}

			s.Equal(tt.expected.out, out.Id)
		})
	}

	// we make sure that all expectations were met
	if err := s.cluster.Mock().ExpectationsWereMet(); err != nil {
		s.T().Errorf("there were unfulfilled expectations: %s", err)
	}
}
