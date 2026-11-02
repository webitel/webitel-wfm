package storage

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
)

// sqlNode records the statements it is given and answers them with one canned
// row, so a test can assert on the SQL a storage method builds.
type sqlNode struct {
	dbsql.Node

	sql  *[]string
	args *[][]any

	// missed makes the update match no row; gone leaves no row to read back.
	missed, gone bool
}

func (n sqlNode) Select(_ context.Context, dest any, sql string, args ...any) error {
	*n.sql = append(*n.sql, sql)
	*n.args = append(*n.args, args)

	if out, ok := dest.(*[]*model.WorkingSchedule); ok && !n.gone {
		*out = []*model.WorkingSchedule{{DomainRecord: model.DomainRecord{ID: 7}}}
	}

	return nil
}

func (n sqlNode) Get(_ context.Context, dest any, sql string, args ...any) error {
	*n.sql = append(*n.sql, sql)
	*n.args = append(*n.args, args)

	if n.missed {
		return dbsql.ParseError(pgx.ErrNoRows)
	}

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
// at the version the client read, bump that version in the same statement, and
// report it whether or not the schedule has extra skills.
func TestUpdateWorkingScheduleIsScopedToOneRow(t *testing.T) {
	for name, skills := range map[string][]*model.LookupItem{
		"without extra skills": nil,
		"with extra skills":    {{ID: 3}},
	} {
		t.Run(name, func(t *testing.T) {
			db, sql, args := newSQLStore()
			w := &WorkingSchedule{db: db}

			user := &model.SignedInUser{ID: 1, DomainID: 2}
			in := &model.WorkingSchedule{DomainRecord: model.DomainRecord{ID: 7}, ExtraSkills: skills, Ver: 4}

			if _, err := w.UpdateWorkingSchedule(userCtx(2, 1), user, in); err != nil {
				t.Fatal(err)
			}

			update, updateArgs := (*sql)[0], (*args)[0]

			scoped := regexp.MustCompile(`UPDATE wfm\.working_schedule SET .*\bver = ver \+ 1 WHERE domain_id = (\$\d+) AND id = (\$\d+) AND ver = (\$\d+) RETURNING id`)

			m := scoped.FindStringSubmatch(update)
			if m == nil {
				t.Fatalf("schedule update is not scoped by domain_id, id and ver, or does not bump ver:\n%s", update)
			}

			if got := argAt(t, updateArgs, m[1]); got != int64(2) {
				t.Errorf("domain_id = %v, want 2", got)
			}

			if got := argAt(t, updateArgs, m[2]); got != int64(7) {
				t.Errorf("id = %v, want 7", got)
			}

			if got := argAt(t, updateArgs, m[3]); got != int32(4) {
				t.Errorf("ver = %v, want 4", got)
			}

			// Tied to what the guard returned, the skills are left alone when it
			// matched nothing; by in.ID they would be wiped, and inserted as orphans.
			if !regexp.MustCompile(`DELETE FROM wfm\.working_schedule_extra_skill WHERE domain_id = \$\d+ AND working_schedule_id IN \(SELECT id FROM schedule\)`).MatchString(update) {
				t.Errorf("skills delete is not tied to the schedule CTE:\n%s", update)
			}

			if len(skills) > 0 && !regexp.MustCompile(`INSERT INTO wfm\.working_schedule_extra_skill \([^)]*\) SELECT .* FROM schedule, `).MatchString(update) {
				t.Errorf("skills insert is not tied to the schedule CTE:\n%s", update)
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

// A guarded update that matched nothing writes nothing more and is resolved by
// rereading the row: still there means another version, gone means NotFound.
func TestUpdateWorkingScheduleMissIsConflictOnlyWhileTheRowExists(t *testing.T) {
	for name, tc := range map[string]struct {
		gone bool
		want error
	}{
		"row at another version": {want: ErrWorkingScheduleVersionConflict},
		"row gone":               {gone: true, want: dbsql.ErrNoRows},
	} {
		t.Run(name, func(t *testing.T) {
			db, sql, _ := newSQLStore()
			db.node.missed, db.node.gone = true, tc.gone
			w := &WorkingSchedule{db: db}

			user := &model.SignedInUser{ID: 1, DomainID: 2}
			in := &model.WorkingSchedule{DomainRecord: model.DomainRecord{ID: 7}, ExtraSkills: []*model.LookupItem{{ID: 3}}, Ver: 4}

			_, err := w.UpdateWorkingSchedule(userCtx(2, 1), user, in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}

			if len(*sql) != 2 || !strings.HasPrefix((*sql)[1], "SELECT ") {
				t.Errorf("want the update and one read, got %q", *sql)
			}
		})
	}
}
