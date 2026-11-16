package migrations

import (
	"context"
	"testing"
)

func TestJobTable(t *testing.T) {
	db, p := newDB(t)
	ctx := context.Background()

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up: %v", err)
	}

	if err := commit(ctx, db, seed, "INSERT INTO directory.wbt_domain (dc, name) VALUES (2, 'other')"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	refused := map[string]string{
		"INSERT INTO wfm.job (kind) VALUES ('planning')":                                                                        "job_planning_schedule_check",
		"INSERT INTO wfm.job (kind, working_schedule_id) VALUES ('planning', 1)":                                                "job_planning_schedule_check",
		"INSERT INTO wfm.job (kind) VALUES ('daily_aggregate')":                                                                 "job_daily_aggregate_scheduled_check",
		"INSERT INTO wfm.job (kind, scheduled_for) VALUES ('report', now())":                                                    "job_kind_check",
		"INSERT INTO wfm.job (kind, scheduled_for, state) VALUES ('daily_aggregate', now(), 'paused')":                          "job_state_check",
		"INSERT INTO wfm.job (kind, scheduled_for, state) VALUES ('daily_aggregate', now(), 'running')":                         "job_lease_check",
		"INSERT INTO wfm.job (kind, scheduled_for, locked_by) VALUES ('daily_aggregate', now(), 'a')":                           "job_lease_check",
		"INSERT INTO wfm.job (kind, scheduled_for, state, locked_by) VALUES ('daily_aggregate', now(), 'running', 'a')":         "job_lease_check",
		"INSERT INTO wfm.job (kind, scheduled_for, state) VALUES ('daily_aggregate', now(), 'failed')":                          "job_failed_error_check",
		"INSERT INTO wfm.job (kind, domain_id, working_schedule_id) VALUES ('planning', 2, 1)":                                  "job_domain_id_working_schedule_id_fkey",
		"INSERT INTO wfm.job (kind, domain_id, working_schedule_id) VALUES ('planning', 1, 1), ('planning', 1, 1)":              "job_schedule_active_udx",
		"INSERT INTO wfm.job (kind, scheduled_for) VALUES ('daily_aggregate', '2026-11-17'), ('daily_aggregate', '2026-11-17')": "job_scheduled_for_udx",
	}

	for stmt, constraint := range refused {
		if got := refusedBy(t, commit(ctx, db, stmt)); got != constraint {
			t.Errorf("%s: refused by %q, want %q", stmt, got, constraint)
		}
	}

	allowed := []string{
		"INSERT INTO wfm.job (kind, domain_id, working_schedule_id, state) VALUES ('planning', 1, 1, 'done'), ('planning', 1, 1, 'done')",
		"INSERT INTO wfm.job (kind, domain_id, working_schedule_id, state, locked_by, locked_until) VALUES ('planning', 1, 1, 'running', 'a', now())",
		"INSERT INTO wfm.job (kind, domain_id, working_schedule_id, state, error) VALUES ('planning', 1, 1, 'failed', 'boom')",
		"INSERT INTO wfm.job (kind, scheduled_for) VALUES ('daily_aggregate', '2026-11-17'), ('daily_aggregate', '2026-11-18')",
	}

	for _, stmt := range allowed {
		if err := commit(ctx, db, stmt); err != nil {
			t.Errorf("%s: %v", stmt, err)
		}
	}

	if err := commit(ctx, db, "DELETE FROM wfm.working_schedule WHERE id = 1"); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM wfm.job WHERE working_schedule_id IS NOT NULL").Scan(&n); err != nil || n != 0 {
		t.Fatalf("%d jobs of a deleted schedule (%v), want 0", n, err)
	}
}
