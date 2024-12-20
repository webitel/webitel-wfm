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

type agentAbsenceTestSuite struct {
	suite.Suite

	log     *wlog.Logger
	cluster *testinfra.TestStorageCluster
	cache   cache.Manager

	store storage.AgentAbsenceManager
}

func TestNewAgentAbsence(t *testing.T) {
	suite.Run(t, new(agentAbsenceTestSuite))
}

func (s *agentAbsenceTestSuite) SetupSuite() {
	var err error

	s.cluster, err = testinfra.NewTestStorageCluster(s.T(), wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: true}))
	if err != nil {
		s.T().Error(err)
	}

	s.cache, err = cache.New(&config.Cache{Size: 1024})
	if err != nil {
		s.T().Error(err)
	}

	s.store = storage.NewAgentAbsence(s.cluster.Store(), s.cache)
}

func (s *agentAbsenceTestSuite) TearDownSuite() {

}

func (s *agentAbsenceTestSuite) TestCreateAgentAbsence() {
	ctx := context.Background()
	user := tests.User(auth_manager.PERMISSION_ACCESS_CREATE.Value(), false)
	type expectation struct {
		out int64
		err error
	}

	// TODO: add more test cases
	t := map[string]struct {
		user     *model.SignedInUser
		in       *model.AgentAbsence
		expected expectation
	}{
		"success": {
			user: user,
			in: &model.AgentAbsence{
				Agent: model.LookupItem{
					Id: 1,
				},
				Absence: model.Absence{
					AbsentAt:    model.NewDate(time.Now().Unix()),
					AbsenceType: 1,
				},
			},
			expected: expectation{
				out: 1,
				err: nil,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			sql := "INSERT INTO wfm.agent_absence (absence_type_id, absent_at, agent_id, created_by, domain_id, updated_by) " +
				"VALUES ($1, $2, $3, $4, $5, $6) RETURNING id"
			args := []any{
				tt.in.Absence.AbsenceType,
				tt.in.Absence.AbsentAt,
				tt.in.Agent.Id,
				tt.user.Id,
				tt.user.DomainId,
				tt.user.Id,
			}

			rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(1))
			s.cluster.Mock().ExpectQuery(sql).WithArgs(args...).WillReturnRows(rows)
			out, err := s.store.CreateAgentAbsence(ctx, tt.user, tt.in)
			s.NoError(err)
			s.Equal(tt.expected.out, out.Absence.Id)
		})
	}

	// we make sure that all expectations were met
	if err := s.cluster.Mock().ExpectationsWereMet(); err != nil {
		s.T().Errorf("there were unfulfilled expectations: %s", err)
	}
}

func (s *agentAbsenceTestSuite) TestUpdateAgentAbsence() {
	ctx := context.Background()
	user := tests.User(auth_manager.PERMISSION_ACCESS_UPDATE.Value(), false)
	type expectation struct {
		out int64
		err error
	}

	// TODO: add more test cases
	t := map[string]struct {
		user     *model.SignedInUser
		in       *model.AgentAbsence
		expected expectation
	}{
		"success": {
			user: user,
			in: &model.AgentAbsence{
				Agent: model.LookupItem{
					Id: 1,
				},
				Absence: model.Absence{
					DomainRecord: model.DomainRecord{
						Id: 1,
					},
					AbsentAt:    model.NewDate(time.Now().Unix()),
					AbsenceType: 1,
				},
			},
			expected: expectation{
				out: 1,
				err: nil,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			sql := "UPDATE wfm.agent_absence SET updated_by = $1, absent_at = $2, absence_type_id = $3 " +
				"WHERE domain_id = $4 AND id = $5 AND agent_id = $6 " +
				"AND ($7 IS FALSE OR EXISTS (SELECT $8 AS rbac FROM wfm.agent_absence_acl " +
				"WHERE dc = $9 AND subject = ANY ($10) AND access & $11 = $12 AND object = $13))"

			args := []any{
				tt.user.Id,                 // 1
				tt.in.Absence.AbsentAt,     // 2
				tt.in.Absence.AbsenceType,  // 3
				tt.user.DomainId,           // 4
				tt.in.Absence.Id,           // 5
				tt.in.Agent.Id,             // 6
				tt.user.UseRBAC,            // 7
				"1",                        // 8
				tt.user.DomainId,           // 9
				tt.user.RbacOptions.Groups, // 10
				tt.user.Access,             // 11
				tt.user.Access,             // 12
				tt.in.Absence.Id,           // 13
			}

			s.cluster.Mock().ExpectExec(sql).WithArgs(args...).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			_, err := s.store.UpdateAgentAbsence(ctx, tt.user, tt.in)
			if err != nil {
				s.T().Error(err)
			}
		})

		// we make sure that all expectations were met
		if err := s.cluster.Mock().ExpectationsWereMet(); err != nil {
			s.T().Errorf("there were unfulfilled expectations: %s", err)
		}
	}
}

func (s *agentAbsenceTestSuite) TestDeleteAgentAbsence() {
	ctx := context.Background()
	user := tests.User(auth_manager.PERMISSION_ACCESS_DELETE.Value(), false)
	type expectation struct {
		out int64
		err error
	}

	// TODO: add more test cases
	t := map[string]struct {
		user     *model.SignedInUser
		agentId  int64
		in       int64
		expected expectation
	}{
		"success": {
			user:    user,
			agentId: 1,
			in:      1,
			expected: expectation{
				out: 1,
				err: nil,
			},
		},
	}

	for scenario, tt := range t {
		s.T().Run(scenario, func(t *testing.T) {
			sql := "DELETE FROM wfm.agent_absence WHERE domain_id = $1 AND id = $2 AND agent_id = $3 " +
				"AND ($4 IS FALSE OR EXISTS (SELECT $5 AS rbac FROM wfm.agent_absence_acl " +
				"WHERE dc = $6 AND subject = ANY ($7) AND access & $8 = $9 AND object = $10))"

			args := []any{
				tt.user.DomainId,           // 1
				tt.in,                      // 2
				tt.agentId,                 // 3
				tt.user.UseRBAC,            // 4
				"1",                        // 5
				tt.user.DomainId,           // 6
				tt.user.RbacOptions.Groups, // 7
				tt.user.Access,             // 8
				tt.user.Access,             // 9
				tt.in,                      // 10
			}

			s.cluster.Mock().ExpectExec(sql).WithArgs(args...).WillReturnResult(pgxmock.NewResult("DELETE", 1))
			if err := s.store.DeleteAgentAbsence(ctx, tt.user, tt.agentId, tt.in); err != nil {
				s.T().Error(err)
			}
		})

		// we make sure that all expectations were met
		if err := s.cluster.Mock().ExpectationsWereMet(); err != nil {
			s.T().Errorf("there were unfulfilled expectations: %s", err)
		}
	}
}
