package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
)

// These tests need a real PostgreSQL and are skipped unless the variable holds
// a URL, e.g. postgres://postgres@127.0.0.1:5432/postgres?sslmode=disable.
const testDSNEnv = "WFM_TEST_POSTGRES_DSN"

// scheduleVer is the migration before the shift model.
const scheduleVer = 20261102120000

// One agent in one schedule.
const seed = `
INSERT INTO directory.wbt_domain (dc, name) VALUES (1, 'domain');
INSERT INTO directory.wbt_user (id, dc, name) VALUES (1, 1, 'user');
INSERT INTO call_center.cc_agent (id, domain_id, user_id) VALUES (1, 1, 1);
INSERT INTO call_center.cc_team (id, domain_id, name) VALUES (1, 1, 'team');
INSERT INTO call_center.cc_skill (id, domain_id, name) VALUES (1, 1, 'skill');
INSERT INTO flow.calendar_timezones (id, sys_name) VALUES (1, 'Europe/Kyiv');
INSERT INTO flow.calendar (id, domain_id, name, timezone_id) VALUES (1, 1, 'calendar', 1);
INSERT INTO wfm.working_schedule (id, domain_id, name, team_id, calendar_id, start_date_at, end_date_at)
VALUES (1, 1, 'schedule', 1, 1, '2026-11-01', '2026-11-30');
INSERT INTO wfm.working_schedule_agent (domain_id, working_schedule_id, agent_id) VALUES (1, 1, 1);`

// newDB creates an empty database with the platform stand-ins, and a provider
// set up as cmd/migrate sets it up.
func newDB(t *testing.T) (*sql.DB, *goose.Provider) {
	t.Helper()

	dsn := os.Getenv(testDSNEnv)
	if dsn == "" {
		t.Skipf("%s is not set", testDSNEnv)
	}

	ctx := context.Background()

	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(func() { _ = admin.Close(ctx) })

	// Random, not time-based: parallel go test runs share the server.
	name := fmt.Sprintf("wfm_migrations_%x", rand.Uint64())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database: %v", err)
	}

	t.Cleanup(func() {
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop database: %v", err)
		}
	})

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}

	cfg.Database = name

	db := stdlib.OpenDB(*cfg)

	t.Cleanup(func() { _ = db.Close() })

	platform, err := os.ReadFile("testdata/platform.sql")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(ctx, string(platform)); err != nil {
		t.Fatalf("platform: %v", err)
	}

	store, err := database.NewStore(database.DialectPostgres, "wfm_schema_version")
	if err != nil {
		t.Fatal(err)
	}

	p, err := goose.NewProvider("", db, Embed, goose.WithStore(store))
	if err != nil {
		t.Fatal(err)
	}

	return db, p
}

// commit runs the statements in one transaction and commits it.
func commit(ctx context.Context, db *sql.DB, stmts ...string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()

			return err
		}
	}

	return tx.Commit()
}

// refusedBy names the constraint that refused err, or its SQLSTATE when the
// error names none; it is empty when err is nil.
func refusedBy(t *testing.T, err error) string {
	t.Helper()

	if err == nil {
		return ""
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("not a database refusal: %v", err)
	}

	if pgErr.ConstraintName != "" {
		return pgErr.ConstraintName
	}

	return pgErr.Code
}

// shift writes the shift with the UTC day of its start as the working date: the
// writer computes the real one, the table only stores it.
func shift(id int, start, end string) string {
	return fmt.Sprintf(`INSERT INTO wfm.agent_working_schedule (id, domain_id, created_by, working_schedule_agent_id, schedule_at, start_at, end_at)
VALUES (%d, 1, 1, 1, '%s', '%s', '%s')`, id, start[:10], start, end)
}

func pause(start, end string) string {
	return fmt.Sprintf(`INSERT INTO wfm.agent_working_schedule_pause (domain_id, agent_working_schedule_id, start_at, end_at)
VALUES (1, 1, '%s', '%s')`, start, end)
}

func TestMigrationsGoDownAndUpAgain(t *testing.T) {
	_, p := newDB(t)
	ctx := context.Background()

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up: %v", err)
	}

	// Down to the last migration of the old model first: going down to 0 drops the
	// whole wfm schema and would hide anything the newer downs leave behind.
	if _, err := p.DownTo(ctx, scheduleVer); err != nil {
		t.Fatalf("down to %d: %v", scheduleVer, err)
	}

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up from %d: %v", scheduleVer, err)
	}

	if _, err := p.DownTo(ctx, 0); err != nil {
		t.Fatalf("down: %v", err)
	}

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

// A database that still has shifts in minutes keeps them: the shift model
// migration fails and leaves the schema as it was.
func TestShiftModelRefusesShiftsInMinutes(t *testing.T) {
	db, p := newDB(t)
	ctx := context.Background()

	if _, err := p.UpTo(ctx, scheduleVer); err != nil {
		t.Fatalf("up to %d: %v", scheduleVer, err)
	}

	old := `INSERT INTO wfm.agent_working_schedule (domain_id, created_by, working_schedule_agent_id, schedule_at, start_min, end_min)
VALUES (1, 1, 1, '2026-11-02', 540, 1080)`
	if err := commit(ctx, db, seed, old); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := p.Up(ctx); err == nil {
		t.Fatal("migrated a table that has shifts in minutes")
	}

	if v, err := p.GetDBVersion(ctx); err != nil || v != scheduleVer {
		t.Fatalf("version = %d (%v), want %d", v, err, scheduleVer)
	}

	var n int
	if err := db.QueryRowContext(ctx, "SELECT count(start_min) FROM wfm.agent_working_schedule").Scan(&n); err != nil || n != 1 {
		t.Fatalf("shifts in minutes = %d (%v), want 1", n, err)
	}
}

func TestShiftModel(t *testing.T) {
	db, p := newDB(t)
	ctx := context.Background()

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up: %v", err)
	}

	if err := commit(ctx, db, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}

	clean := func(t *testing.T) {
		t.Helper()

		t.Cleanup(func() {
			if err := commit(ctx, db, "DELETE FROM wfm.agent_working_schedule"); err != nil {
				t.Errorf("clean up: %v", err)
			}
		})
	}

	day := shift(1, "2026-11-02 08:00Z", "2026-11-02 16:00Z")
	lunch := pause("2026-11-02 12:00Z", "2026-11-02 13:00Z")
	skill := "INSERT INTO wfm.agent_working_schedule_skill (domain_id, agent_working_schedule_id, skill_id) VALUES (1, 1, 1)"

	cases := []struct {
		name  string
		given []string // committed before stmts
		stmts []string
		want  string // the constraint that refuses the transaction, empty when it commits
	}{
		{
			name:  "a shift of 24 hours",
			stmts: []string{shift(1, "2026-11-02 08:00Z", "2026-11-03 08:00Z")},
		},
		{
			name:  "a shift over 24 hours",
			stmts: []string{shift(1, "2026-11-02 08:00Z", "2026-11-03 08:00:01Z")},
			want:  "agent_working_schedule_duration_check",
		},
		{
			name:  "a shift that ends as it starts",
			stmts: []string{shift(1, "2026-11-02 08:00Z", "2026-11-02 08:00Z")},
			want:  "agent_working_schedule_duration_check",
		},
		{
			name:  "two shifts of one date, back to back",
			stmts: []string{shift(1, "2026-11-02 07:00Z", "2026-11-02 11:00Z"), shift(2, "2026-11-02 11:00Z", "2026-11-02 15:00Z")},
		},
		{
			name: "a shift without a working date",
			stmts: []string{`INSERT INTO wfm.agent_working_schedule (domain_id, created_by, working_schedule_agent_id, start_at, end_at)
VALUES (1, 1, 1, '2026-11-02 08:00Z', '2026-11-02 16:00Z')`},
			want: pgerrcode.NotNullViolation,
		},
		{
			name:  "a pause up to the end of its shift",
			stmts: []string{day, pause("2026-11-02 15:00Z", "2026-11-02 16:00Z")},
		},
		{
			name:  "a pause before its shift starts",
			stmts: []string{day, pause("2026-11-02 07:30Z", "2026-11-02 08:30Z")},
			want:  "agent_working_schedule_pause_inside",
		},
		{
			name:  "a pause that ends as it starts",
			stmts: []string{day, pause("2026-11-02 12:00Z", "2026-11-02 12:00Z")},
			want:  "agent_working_schedule_pause_duration_check",
		},
		{
			name:  "a shift cut short of its pause",
			given: []string{day, lunch},
			stmts: []string{"UPDATE wfm.agent_working_schedule SET end_at = '2026-11-02 12:30Z'"},
			want:  "agent_working_schedule_pause_inside",
		},
		{
			name:  "a shift moved away from its pause, then the pause after it",
			given: []string{day, lunch},
			stmts: []string{
				"UPDATE wfm.agent_working_schedule SET start_at = '2026-11-02 18:00Z', end_at = '2026-11-03 02:00Z'",
				"UPDATE wfm.agent_working_schedule_pause SET start_at = '2026-11-02 20:00Z', end_at = '2026-11-02 21:00Z'",
			},
		},
		{
			name:  "a skill without a capacity",
			stmts: []string{day, skill},
		},
		{
			name:  "a skill twice in one shift",
			stmts: []string{day, skill, skill},
			want:  "agent_working_schedule_skill_shift_skill_udx",
		},
		{
			name:  "an agent twice in one schedule",
			stmts: []string{"INSERT INTO wfm.working_schedule_agent (domain_id, working_schedule_id, agent_id) VALUES (1, 1, 1)"},
			want:  "working_schedule_agent_schedule_agent_udx",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clean(t)

			if err := commit(ctx, db, tc.given...); err != nil {
				t.Fatalf("given: %v", err)
			}

			if got := refusedBy(t, commit(ctx, db, tc.stmts...)); got != tc.want {
				t.Errorf("refused by %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("a new shift is version 1, written by hand", func(t *testing.T) {
		clean(t)

		if err := commit(ctx, db, day); err != nil {
			t.Fatal(err)
		}

		var ver, source int
		if err := db.QueryRowContext(ctx, "SELECT ver, source FROM wfm.agent_working_schedule").Scan(&ver, &source); err != nil {
			t.Fatal(err)
		}

		if ver != 1 || source != 1 {
			t.Errorf("ver %d, source %d, want 1 and 1 (manual)", ver, source)
		}
	})

	// Pauses and skills are read, checked and deleted by their shift.
	t.Run("pauses and skills are indexed by their shift", func(t *testing.T) {
		for _, table := range []string{"wfm.agent_working_schedule_pause", "wfm.agent_working_schedule_skill"} {
			var ok bool

			err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT
               FROM pg_index i
                        JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = i.indkey[0]
               WHERE i.indrelid = $1::REGCLASS
                 AND a.attname = 'agent_working_schedule_id')`, table).Scan(&ok)
			if err != nil {
				t.Fatal(err)
			}

			if !ok {
				t.Errorf("%s has no index that starts with agent_working_schedule_id", table)
			}
		}
	})
}

// An agent saved twice in one schedule, which the code allowed until now, is
// kept once, as its oldest row.
func TestShiftModelDropsDuplicateAgents(t *testing.T) {
	db, p := newDB(t)
	ctx := context.Background()

	if _, err := p.UpTo(ctx, scheduleVer); err != nil {
		t.Fatalf("up to %d: %v", scheduleVer, err)
	}

	again := "INSERT INTO wfm.working_schedule_agent (domain_id, working_schedule_id, agent_id) VALUES (1, 1, 1)"
	if err := commit(ctx, db, seed, again); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up: %v", err)
	}

	var n, id int
	if err := db.QueryRowContext(ctx, "SELECT count(*), min(id) FROM wfm.working_schedule_agent").Scan(&n, &id); err != nil {
		t.Fatal(err)
	}

	if n != 1 || id != 1 {
		t.Errorf("rows = %d, first id = %d; want the agent once, as row 1", n, id)
	}
}
