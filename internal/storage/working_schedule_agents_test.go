package storage

import (
	"strings"
	"testing"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
)

// An agent is in a schedule once, so adding one the schedule already has, or the
// same one twice in a request, must not fail on that key.
func TestAgentInsertsSkipAnAgentAlreadyInTheSchedule(t *testing.T) {
	rows := schedule(7, "schedule")
	ws, tr := newProbe(t, &rows, &rows, 7)
	ctx := userCtx(1, 1)

	user := &model.SignedInUser{ID: 1, DomainID: 1}
	if _, err := ws.CreateWorkingSchedule(ctx, user, &model.WorkingSchedule{Agents: []*model.LookupItem{{ID: 3}, {ID: 3}}}); err != nil {
		t.Fatal(err)
	}

	read, err := options.NewRead(ctx, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ws.UpdateWorkingScheduleAddAgents(ctx, read, []int64{3}); err != nil {
		t.Fatal(err)
	}

	var inserts int

	for _, c := range tr.calls {
		for _, stmt := range strings.Split(c.sql, "INSERT INTO ")[1:] {
			if !strings.HasPrefix(stmt, builder.WorkingScheduleAgentTable.Name()+" ") {
				continue
			}

			inserts++

			if !strings.Contains(stmt, "ON CONFLICT DO NOTHING") {
				t.Errorf("an agent insert fails on an agent the schedule already has:\n%s", c.sql)
			}
		}
	}

	if inserts != 2 {
		t.Fatalf("agent inserts = %d, want 2: on create and on add", inserts)
	}
}
