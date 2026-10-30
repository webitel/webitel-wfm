package storage

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
)

// sqlNode records the statements it is given and answers them with one canned
// row, so a test can assert on the SQL a storage method builds.
type sqlNode struct {
	dbsql.Node

	sql  *[]string
	args *[][]any
}

func (n sqlNode) Select(_ context.Context, dest any, sql string, args ...any) error {
	*n.sql = append(*n.sql, sql)
	*n.args = append(*n.args, args)

	if out, ok := dest.(*[]*model.WorkingSchedule); ok {
		*out = []*model.WorkingSchedule{{DomainRecord: model.DomainRecord{ID: 7}}}
	}

	return nil
}

func (n sqlNode) Get(_ context.Context, dest any, sql string, args ...any) error {
	*n.sql = append(*n.sql, sql)
	*n.args = append(*n.args, args)

	if out, ok := dest.(*int64); ok {
		*out = 7
	}

	return nil
}

type sqlStore struct {
	dbsql.Store

	node sqlNode
}

func (s *sqlStore) Primary() dbsql.Node          { return s.node }
func (s *sqlStore) StandbyPreferred() dbsql.Node { return s.node }

func newSQLStore() (*sqlStore, *[]string, *[][]any) {
	sql, args := &[]string{}, &[][]any{}

	return &sqlStore{node: sqlNode{sql: sql, args: args}}, sql, args
}

// An update must touch exactly the schedule it names, in the caller's domain,
// and report it whether or not the schedule has extra skills.
func TestUpdateWorkingScheduleIsScopedToOneRow(t *testing.T) {
	for name, skills := range map[string][]*model.LookupItem{
		"without extra skills": nil,
		"with extra skills":    {{ID: 3}},
	} {
		t.Run(name, func(t *testing.T) {
			db, sql, args := newSQLStore()
			w := &WorkingSchedule{db: db}

			user := &model.SignedInUser{ID: 1, DomainID: 2}
			in := &model.WorkingSchedule{DomainRecord: model.DomainRecord{ID: 7}, ExtraSkills: skills}

			if _, err := w.UpdateWorkingSchedule(userCtx(2, 1), user, in); err != nil {
				t.Fatal(err)
			}

			update, updateArgs := (*sql)[0], (*args)[0]

			scoped := regexp.MustCompile(`UPDATE wfm\.working_schedule SET .* WHERE domain_id = (\$\d+) AND id = (\$\d+) RETURNING id`)

			m := scoped.FindStringSubmatch(update)
			if m == nil {
				t.Fatalf("schedule update is not scoped by domain_id and id:\n%s", update)
			}

			if got := argAt(t, updateArgs, m[1]); got != int64(2) {
				t.Errorf("domain_id = %v, want 2", got)
			}

			if got := argAt(t, updateArgs, m[2]); got != int64(7) {
				t.Errorf("id = %v, want 7", got)
			}

			// A cross join with the other CTEs yields no row when one of them
			// is empty, which turns a successful update into NotFound.
			if !regexp.MustCompile(`SELECT schedule\.id FROM schedule$`).MatchString(update) {
				t.Errorf("result must be read from the schedule CTE alone:\n%s", update)
			}
		})
	}
}

func argAt(t *testing.T, args []any, placeholder string) any {
	t.Helper()

	var i int
	if _, err := fmt.Sscanf(placeholder, "$%d", &i); err != nil || i < 1 || i > len(args) {
		t.Fatalf("placeholder %s out of range of %d args", placeholder, len(args))
	}

	return args[i-1]
}
