package storage

import (
	"testing"

	"github.com/webitel/webitel-wfm/internal/model"
)

func TestOptionalLookupsOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, shiftSeed+`
INSERT INTO wfm.pause_template (id, domain_id, created_by, updated_by, name) VALUES (1, 1, 1, 1, 'pauses');
INSERT INTO wfm.shift_template (id, domain_id, created_by, updated_by, name, times) VALUES (1, 1, 1, 1, 'shifts', '[]');
INSERT INTO wfm.working_condition (id, domain_id, created_by, updated_by, name, pause_template_id, shift_template_id)
VALUES (1, 1, 1, 1, 'without a shift template', 1, NULL),
       (2, 1, 1, 1, 'with a shift template', 1, 1);
INSERT INTO wfm.agent_working_conditions (domain_id, updated_by, agent_id, working_condition_id, pause_template_id)
VALUES (1, 1, 1, 1, NULL),
       (1, 1, 2, 1, 1);`)
	ctx := userCtx(1, 1)

	lookup := func(l *model.LookupItem) string {
		if l == nil {
			return "none"
		}

		if l.Name == nil {
			return "no name"
		}

		return *l.Name
	}

	for id, want := range map[int64]string{1: "none", 2: "shifts"} {
		wc, err := NewWorkingCondition(db).ReadWorkingCondition(ctx, readOf(t, ctx, id))
		if err != nil {
			t.Fatal(err)
		}

		if got := lookup(wc.ShiftTemplate); got != want {
			t.Errorf("working condition %d: shift template %q, want %q", id, got, want)
		}
	}

	for agent, want := range map[int64]string{1: "none", 2: "pauses"} {
		awc, err := NewAgentWorkingConditions(db).ReadAgentWorkingConditions(ctx, readOf(t, ctx, agent))
		if err != nil {
			t.Fatal(err)
		}

		if got := lookup(awc.PauseTemplate); got != want {
			t.Errorf("agent %d: pause template %q, want %q", agent, got, want)
		}
	}
}
