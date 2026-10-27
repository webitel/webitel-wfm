package dbsql

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/infra/pgw"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// These tests need a real PostgreSQL and are skipped unless the variable holds
// a URL, e.g. postgres://postgres:pg@127.0.0.1:55432/postgres?sslmode=disable.
const testDSNEnv = "WFM_TEST_POSTGRES_DSN"

func pgSchema(t *testing.T, ddl ...string) string {
	t.Helper()

	dsn := os.Getenv(testDSNEnv)
	if dsn == "" {
		t.Skipf("%s is not set", testDSNEnv)
	}

	// Random, not time-based: parallel go test runs share the database.
	schema := fmt.Sprintf("dbsql_tx_%x", rand.Uint64())
	admin := func(stmts ...string) {
		t.Helper()

		ctx := context.Background()

		conn, err := pgx.Connect(ctx, dsn)
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		defer conn.Close(ctx)

		for _, stmt := range stmts {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				t.Fatalf("%s: %v", stmt, err)
			}
		}
	}

	admin(append([]string{"CREATE SCHEMA " + schema, "SET search_path TO " + schema}, ddl...)...)
	t.Cleanup(func() { admin("DROP SCHEMA " + schema + " CASCADE") })

	return withParam(t, dsn, "search_path", schema)
}

func withParam(t *testing.T, dsn, key, value string) string {
	t.Helper()

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("%s must be a URL: %v", testDSNEnv, err)
	}

	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()

	return u.String()
}

func connect(t *testing.T, dsn string, maxConns int) *DB {
	t.Helper()

	pm, err := pgw.NewPoolManager(context.Background(), pgw.WithPrimaryConfig(pgw.PrimaryConfig{DSN: dsn, MaxConns: maxConns}))
	if err != nil {
		t.Fatalf("pool manager: %v", err)
	}

	db := New(pm)

	t.Cleanup(func() { _ = db.Close() })

	return db
}

func count(t *testing.T, db Store, table string) int64 {
	t.Helper()

	var n int64
	if err := db.Primary().Get(context.Background(), &n, "SELECT count(*) FROM "+table); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	return n
}

// With a single-connection pool, a changed pid means the connection was destroyed.
func backendPID(t *testing.T, node Node) int64 {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var pid int64
	if err := node.Get(ctx, &pid, "SELECT pg_backend_pid()"); err != nil {
		t.Fatalf("pg_backend_pid: %v", err)
	}

	return pid
}

func TestPGInTxIsAllOrNothing(t *testing.T) {
	dsn := pgSchema(t,
		"CREATE TABLE item (id int PRIMARY KEY)",
		"CREATE TABLE deferred_item (id int, CONSTRAINT deferred_item_id_key UNIQUE (id) DEFERRABLE INITIALLY DEFERRED)",
	)
	db := connect(t, dsn, 4)

	insert := func(ctx context.Context, tx Store, ids ...int) error {
		for _, id := range ids {
			if err := tx.Primary().Exec(ctx, "INSERT INTO item (id) VALUES ($1)", id); err != nil {
				return err
			}
		}

		return nil
	}

	tests := map[string]struct {
		seed     string
		fn       func(ctx context.Context, tx Store) error
		table    string
		wantErr  error
		sqlState string
		wantRows int64
	}{
		"nil commits every step": {
			fn:       func(ctx context.Context, tx Store) error { return insert(ctx, tx, 1, 2, 3) },
			table:    "item",
			wantRows: 3,
		},
		"an error from fn undoes earlier steps": {
			fn: func(ctx context.Context, tx Store) error {
				if err := insert(ctx, tx, 1, 2); err != nil {
					return err
				}

				return errRefused
			},
			table:   "item",
			wantErr: errRefused,
		},
		"a failing statement undoes earlier steps": {
			fn:       func(ctx context.Context, tx Store) error { return insert(ctx, tx, 1, 2, 1) },
			table:    "item",
			wantErr:  ErrUniqueViolation,
			sqlState: pgerrcode.UniqueViolation,
		},
		"a batch stays inside the transaction": {
			fn: func(ctx context.Context, tx Store) error {
				b := tx.Primary().Batch()
				for id := range 3 {
					b.Queue("INSERT INTO item (id) VALUES ($1) RETURNING id", id)
				}

				var ids []int64
				if err := b.Select(ctx, &ids); err != nil {
					return err
				}

				return errRefused
			},
			table:   "item",
			wantErr: errRefused,
		},
		"a failing statement inside a batch undoes earlier steps": {
			fn: func(ctx context.Context, tx Store) error {
				if err := insert(ctx, tx, 1); err != nil {
					return err
				}

				b := tx.Primary().Batch()
				b.Queue("INSERT INTO item (id) VALUES (2) RETURNING id")
				b.Queue("INSERT INTO item (id) VALUES (1) RETURNING id")
				b.Queue("INSERT INTO item (id) VALUES (3) RETURNING id")

				var ids []int64

				return b.Select(ctx, &ids)
			},
			table:    "item",
			wantErr:  ErrUniqueViolation,
			sqlState: pgerrcode.UniqueViolation,
		},
		"a conditional update that matches no row undoes earlier steps": {
			fn: func(ctx context.Context, tx Store) error {
				if err := insert(ctx, tx, 1); err != nil {
					return err
				}

				var id int64

				return tx.Primary().Get(ctx, &id, "UPDATE item SET id = id WHERE id = $1 RETURNING id", 99)
			},
			table:   "item",
			wantErr: ErrNoRows,
		},
		"a failing select undoes earlier steps": {
			fn: func(ctx context.Context, tx Store) error {
				if err := insert(ctx, tx, 1); err != nil {
					return err
				}

				var ids []int64

				return tx.Primary().Select(ctx, &ids, "INSERT INTO item (id) VALUES (2), (1) RETURNING id")
			},
			table:    "item",
			wantErr:  ErrUniqueViolation,
			sqlState: pgerrcode.UniqueViolation,
		},
		"a failed statement poisons the transaction even when fn swallows it": {
			fn: func(ctx context.Context, tx Store) error {
				if err := insert(ctx, tx, 1); err != nil {
					return err
				}

				_ = insert(ctx, tx, 1)

				return nil
			},
			table:   "item",
			wantErr: pgx.ErrTxCommitRollback,
		},
		"deleting existing rows is undone too": {
			seed: "INSERT INTO item (id) VALUES (1), (2), (3)",
			fn: func(ctx context.Context, tx Store) error {
				if err := tx.Primary().Exec(ctx, "DELETE FROM item"); err != nil {
					return err
				}

				if err := insert(ctx, tx, 7, 8); err != nil {
					return err
				}

				return errRefused
			},
			table:    "item",
			wantErr:  errRefused,
			wantRows: 3,
		},
		"a constraint checked at COMMIT fails the whole transaction": {
			fn: func(ctx context.Context, tx Store) error {
				if err := tx.Primary().Exec(ctx, "INSERT INTO deferred_item (id) VALUES (1), (2)"); err != nil {
					return err
				}

				return tx.Primary().Exec(ctx, "INSERT INTO deferred_item (id) VALUES (1)")
			},
			table:    "deferred_item",
			wantErr:  ErrUniqueViolation,
			sqlState: pgerrcode.UniqueViolation,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			t.Cleanup(func() {
				if err := db.Primary().Exec(ctx, "TRUNCATE item, deferred_item"); err != nil {
					t.Fatalf("truncate: %v", err)
				}
			})

			if tc.seed != "" {
				if err := db.Primary().Exec(ctx, tc.seed); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}

			err := db.InTx(ctx, tc.fn)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("InTx = %v, want %v", err, tc.wantErr)
			}

			if tc.wantErr != nil && errors.Code(err) != errors.Code(tc.wantErr) {
				t.Errorf("code = %v, want %v", errors.Code(err), errors.Code(tc.wantErr))
			}

			var pgErr *pgconn.PgError
			if tc.sqlState != "" && (!errors.As(err, &pgErr) || pgErr.Code != tc.sqlState) {
				t.Errorf("SQLSTATE behind %v = %v, want %s", err, pgErr, tc.sqlState)
			}

			if rbErr, failed := errors.Lookup(err, "rollback"); failed {
				t.Errorf("ROLLBACK failed: %v", rbErr)
			}

			if got := count(t, db, tc.table); got != tc.wantRows {
				t.Errorf("%s has %d rows after InTx, want %d", tc.table, got, tc.wantRows)
			}
		})
	}
}

func TestPGInTxIsReadCommitted(t *testing.T) {
	dsn := pgSchema(t, "CREATE TABLE item (id int PRIMARY KEY)")
	db := connect(t, withParam(t, dsn, "default_transaction_isolation", "serializable"), 4)
	ctx := context.Background()

	var level string
	if err := db.Primary().Get(ctx, &level, "SHOW transaction_isolation"); err != nil || level != "serializable" {
		t.Fatalf("outside InTx: level = %q (%v), want the serializable default to be in effect", level, err)
	}

	err := db.InTx(ctx, func(ctx context.Context, tx Store) error {
		if err := tx.Primary().Get(ctx, &level, "SHOW transaction_isolation"); err != nil {
			return err
		}

		if level != "read committed" {
			t.Errorf("inside InTx: level = %q, want read committed", level)
		}

		if err := tx.Primary().Exec(ctx, "INSERT INTO item (id) VALUES (1)"); err != nil {
			return err
		}

		if n := count(t, db, "item"); n != 0 {
			t.Errorf("another session sees %d uncommitted rows, want 0", n)
		}

		if err := db.Primary().Exec(ctx, "INSERT INTO item (id) VALUES (2)"); err != nil {
			return err
		}

		if n := count(t, tx, "item"); n != 2 {
			t.Errorf("inside InTx sees %d rows after a concurrent commit, want 2", n)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("InTx = %v", err)
	}
}

func TestPGInTxRunsOnThePrimaryOnly(t *testing.T) {
	dsn := pgSchema(t, "CREATE TABLE item (id int PRIMARY KEY)")

	// The standby is the same server with read-only sessions: writes through it fail with 25006.
	pm, err := pgw.NewPoolManager(context.Background(),
		pgw.WithPrimaryConfig(pgw.PrimaryConfig{DSN: dsn, MaxConns: 2}),
		pgw.WithStandbyConfig(pgw.StandbyConfig{DSN: []string{withParam(t, dsn, "default_transaction_read_only", "on")}, MaxConns: 2}),
	)
	if err != nil {
		t.Fatalf("pool manager: %v", err)
	}

	db := New(pm)

	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		if _, err := pm.Standby(); err == nil {
			break
		}

		select {
		case <-ctx.Done():
			t.Fatal("the standby never connected")
		case <-time.After(50 * time.Millisecond):
		}
	}

	var pgErr *pgconn.PgError
	if err := db.StandbyPreferred().Exec(ctx, "INSERT INTO item (id) VALUES (0)"); !errors.As(err, &pgErr) || pgErr.Code != pgerrcode.ReadOnlySQLTransaction {
		t.Fatalf("a write through the standby = %v, want 25006; without it this test proves nothing", err)
	}

	err = db.InTx(ctx, func(ctx context.Context, tx Store) error {
		for id, node := range []Node{tx.Primary(), tx.StandbyPreferred()} {
			if err := node.Exec(ctx, "INSERT INTO item (id) VALUES ($1)", id+1); err != nil {
				return err
			}
		}

		var n int64
		if err := tx.StandbyPreferred().Get(ctx, &n, "SELECT count(*) FROM item"); err != nil {
			return err
		}

		if n != 2 {
			t.Errorf("the standby role inside InTx reads %d rows, want its own 2", n)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("InTx = %v", err)
	}

	if n := count(t, db, "item"); n != 2 {
		t.Errorf("%d rows committed, want 2", n)
	}
}

func TestPGInTxJoins(t *testing.T) {
	dsn := pgSchema(t, "CREATE TABLE item (id int PRIMARY KEY)")
	db := connect(t, dsn, 4)

	truncate := func(t *testing.T) {
		t.Helper()

		t.Cleanup(func() {
			if err := db.Primary().Exec(context.Background(), "TRUNCATE item"); err != nil {
				t.Fatalf("truncate: %v", err)
			}
		})
	}

	t.Run("every path runs in the one transaction", func(t *testing.T) {
		truncate(t)

		var outer, standby, inner, batched []int64

		err := db.InTx(context.Background(), func(ctx context.Context, tx Store) error {
			if err := tx.Primary().Select(ctx, &outer, "SELECT txid_current()"); err != nil {
				return err
			}

			if err := tx.Primary().Exec(ctx, "INSERT INTO item (id) VALUES (1)"); err != nil {
				return err
			}

			return tx.InTx(ctx, func(ctx context.Context, tx Store) error {
				if err := tx.StandbyPreferred().Select(ctx, &standby, "SELECT txid_current()"); err != nil {
					return err
				}

				if err := tx.Primary().Select(ctx, &inner, "SELECT txid_current()"); err != nil {
					return err
				}

				b := tx.Primary().Batch()
				b.Queue("SELECT txid_current()")
				b.Queue("INSERT INTO item (id) VALUES (2) RETURNING 0::bigint")

				if err := b.Select(ctx, &batched); err != nil {
					return err
				}

				return errRefused
			})
		})
		if !errors.Is(err, errRefused) {
			t.Fatalf("InTx = %v, want the callee's error", err)
		}

		for name, got := range map[string][]int64{"standby": standby, "nested": inner, "batch": batched[:1]} {
			if len(got) != 1 || got[0] != outer[0] {
				t.Errorf("%s ran in transaction %v, want %v", name, got, outer)
			}
		}

		if n := count(t, db, "item"); n != 0 {
			t.Errorf("%d rows survived the callee's error, want 0", n)
		}
	})

	t.Run("a callee's nil is not a commit", func(t *testing.T) {
		truncate(t)

		err := db.InTx(context.Background(), func(ctx context.Context, tx Store) error {
			err := tx.InTx(ctx, func(ctx context.Context, tx Store) error {
				return tx.Primary().Exec(ctx, "INSERT INTO item (id) VALUES (1)")
			})
			if err != nil {
				return err
			}

			return errRefused
		})
		if !errors.Is(err, errRefused) {
			t.Fatalf("InTx = %v, want the caller's error", err)
		}

		if n := count(t, db, "item"); n != 0 {
			t.Errorf("the callee's row survived the caller's error (%d rows), want 0", n)
		}
	})

	t.Run("a swallowed callee error does not undo the callee's writes", func(t *testing.T) {
		truncate(t)

		err := db.InTx(context.Background(), func(ctx context.Context, tx Store) error {
			if err := tx.Primary().Exec(ctx, "INSERT INTO item (id) VALUES (1)"); err != nil {
				return err
			}

			_ = tx.InTx(ctx, func(ctx context.Context, tx Store) error {
				if err := tx.Primary().Exec(ctx, "INSERT INTO item (id) VALUES (2)"); err != nil {
					return err
				}

				return errRefused
			})

			return nil
		})
		if err != nil {
			t.Fatalf("InTx = %v", err)
		}

		if n := count(t, db, "item"); n != 2 {
			t.Errorf("%d rows committed, want 2: a joined InTx must not act as a savepoint", n)
		}
	})
}

func TestPGInTxKeepsThePoolHealthy(t *testing.T) {
	dsn := pgSchema(t, "CREATE TABLE item (id int PRIMARY KEY)")
	db := connect(t, dsn, 1)

	t.Run("ctx canceled after the last statement", func(t *testing.T) {
		before := backendPID(t, db.Primary())
		ctx, cancel := context.WithCancel(context.Background())

		err := db.InTx(ctx, func(ctx context.Context, tx Store) error {
			if err := tx.Primary().Exec(ctx, "INSERT INTO item (id) VALUES (1)"); err != nil {
				return err
			}

			cancel()

			return ctx.Err()
		})
		if err == nil {
			t.Fatal("InTx returned no error")
		}

		if after := backendPID(t, db.Primary()); after != before {
			t.Errorf("the connection was replaced (pid %d → %d): ROLLBACK did not reach the server", before, after)
		}

		if n := count(t, db, "item"); n != 0 {
			t.Errorf("%d rows survived the cancel, want 0", n)
		}
	})

	t.Run("ctx canceled after fn returned nil", func(t *testing.T) {
		before := backendPID(t, db.Primary())
		ctx, cancel := context.WithCancel(context.Background())

		err := db.InTx(ctx, func(ctx context.Context, tx Store) error {
			if err := tx.Primary().Exec(ctx, "INSERT INTO item (id) VALUES (2)"); err != nil {
				return err
			}

			cancel()

			return nil
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("InTx = %v, want context.Canceled as the cause", err)
		}

		if after := backendPID(t, db.Primary()); after != before {
			t.Errorf("the connection was replaced (pid %d → %d)", before, after)
		}

		if n := count(t, db, "item"); n != 0 {
			t.Errorf("%d rows committed by a canceled request, want 0", n)
		}
	})

	t.Run("ctx canceled mid-statement", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()

		err := db.InTx(ctx, func(ctx context.Context, tx Store) error {
			if err := tx.Primary().Exec(ctx, "INSERT INTO item (id) VALUES (3)"); err != nil {
				return err
			}

			return tx.Primary().Exec(ctx, "SELECT pg_sleep(5)")
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("InTx = %v, want context.DeadlineExceeded as the cause", err)
		}

		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("InTx returned after %v; the statement ignored the canceled request", elapsed)
		}

		backendPID(t, db.Primary())

		if n := count(t, db, "item"); n != 0 {
			t.Errorf("%d rows survived the cancel, want 0", n)
		}
	})

	t.Run("panic on a canceled ctx", func(t *testing.T) {
		before := backendPID(t, db.Primary())
		ctx, cancel := context.WithCancel(context.Background())

		func() {
			defer func() { _ = recover() }()

			_ = db.InTx(ctx, func(ctx context.Context, tx Store) error {
				if err := tx.Primary().Exec(ctx, "INSERT INTO item (id) VALUES (4)"); err != nil {
					return err
				}

				cancel()
				panic("boom")
			})
		}()

		if after := backendPID(t, db.Primary()); after != before {
			t.Errorf("the connection was replaced (pid %d → %d): ROLLBACK did not reach the server", before, after)
		}

		if n := count(t, db, "item"); n != 0 {
			t.Errorf("%d rows survived the panic, want 0", n)
		}
	})

	t.Run("waiting for a connection honors ctx", func(t *testing.T) {
		held, release, holder := make(chan struct{}), make(chan struct{}), make(chan error, 1)

		go func() {
			holder <- db.InTx(context.Background(), func(context.Context, Store) error {
				close(held)
				<-release

				return nil
			})
		}()

		<-held

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		called, waiter := false, make(chan error, 1)

		go func() {
			waiter <- db.InTx(ctx, func(context.Context, Store) error {
				called = true

				return nil
			})
		}()

		select {
		case err := <-waiter:
			if !errors.Is(err, ErrInternal) || !errors.Is(err, context.DeadlineExceeded) || called {
				t.Errorf("InTx = %v, fn called = %v; want ErrInternal caused by the deadline and fn not run", err, called)
			}
		case <-time.After(3 * time.Second):
			t.Error("InTx kept waiting for a connection after its ctx ended")
		}

		close(release)

		if err := <-holder; err != nil {
			t.Errorf("holder InTx = %v", err)
		}
	})
}

const (
	shiftDDL = `CREATE TABLE shift (
		id       bigserial PRIMARY KEY,
		agent_id int         NOT NULL,
		start_at timestamptz NOT NULL,
		end_at   timestamptz NOT NULL
	)`

	shiftLockClass = 80
)

var errOverlap = errors.New("the shift overlaps another shift of the operator", errors.WithCode(codes.FailedPrecondition))

// book is the WFM-94 check-then-write; gap runs between the check and the write.
func book(ctx context.Context, db Store, agent int, start time.Time, lock bool, gap func(context.Context) error) error {
	end := start.Add(8 * time.Hour)

	return db.InTx(ctx, func(ctx context.Context, tx Store) error {
		if lock {
			if err := tx.Primary().Exec(ctx, "SELECT pg_advisory_xact_lock($1, $2)", shiftLockClass, agent); err != nil {
				return err
			}
		}

		var overlaps int64

		err := tx.Primary().Get(ctx, &overlaps, `SELECT count(*) FROM shift
			WHERE agent_id = $1 AND tstzrange(start_at, end_at) && tstzrange($2, $3)`, agent, start, end)
		if err != nil {
			return err
		}

		if overlaps > 0 {
			return errOverlap
		}

		if err := gap(ctx); err != nil {
			return err
		}

		return tx.Primary().Exec(ctx, "INSERT INTO shift (agent_id, start_at, end_at) VALUES ($1, $2, $3)", agent, start, end)
	})
}

func shiftsPerAgent(t *testing.T, db Store) map[int]int64 {
	t.Helper()

	var rows []struct {
		AgentID int
		N       int64
	}
	if err := db.Primary().Select(context.Background(), &rows, "SELECT agent_id, count(*) AS n FROM shift GROUP BY agent_id"); err != nil {
		t.Fatalf("shifts per agent: %v", err)
	}

	got := map[int]int64{}
	for _, r := range rows {
		got[r.AgentID] = r.N
	}

	return got
}

// Control for the next test: without a lock every writer passes the check.
func TestPGCheckThenWriteRacesWithoutALock(t *testing.T) {
	const writers = 8

	dsn := pgSchema(t, shiftDDL)
	db := connect(t, dsn, writers)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	arrived, release := make(chan struct{}, writers), make(chan struct{})
	go func() {
		for range writers {
			<-arrived
		}

		close(release)
	}()

	barrier := func(ctx context.Context) error {
		arrived <- struct{}{}

		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	start := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			if err := book(ctx, db, 1, start, false, barrier); err != nil {
				t.Errorf("book: %v", err)
			}
		})
	}

	wg.Wait()

	if got := shiftsPerAgent(t, db)[1]; got != writers {
		t.Errorf("operator has %d overlapping shifts, want all %d writers to have raced through", got, writers)
	}
}

func TestPGInTxSerializesWritersUnderAnAdvisoryLock(t *testing.T) {
	const (
		agents          = 3
		writersPerAgent = 20
	)

	dsn := pgSchema(t, shiftDDL)
	instances := []*DB{connect(t, dsn, 8), connect(t, dsn, 8)}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Widens the race window, so a missing lock would show up.
	gap := func(context.Context) error {
		time.Sleep(5 * time.Millisecond)

		return nil
	}

	day := time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)

	var (
		wg                 sync.WaitGroup
		mu                 sync.Mutex
		booked, refused    int
		unexpected         []error
		refusedCodeChanged bool
	)

	for agent := 1; agent <= agents; agent++ {
		for w := range writersPerAgent {
			wg.Go(func() {
				// Starts within 4h, shifts 8h long: any two of an operator's overlap.
				start := day.Add(time.Duration(rand.IntN(240)) * time.Minute)

				err := book(ctx, instances[w%len(instances)], agent, start, true, gap)

				mu.Lock()
				defer mu.Unlock()

				switch {
				case err == nil:
					booked++
				case errors.Is(err, errOverlap):
					refused++
					refusedCodeChanged = refusedCodeChanged || errors.Code(err) != codes.FailedPrecondition
				default:
					unexpected = append(unexpected, err)
				}
			})
		}
	}

	wg.Wait()

	if len(unexpected) > 0 {
		t.Fatalf("%d writers failed unexpectedly, first: %v", len(unexpected), unexpected[0])
	}

	if booked != agents || refused != agents*(writersPerAgent-1) {
		t.Errorf("booked %d and refused %d, want %d and %d", booked, refused, agents, agents*(writersPerAgent-1))
	}

	if refusedCodeChanged {
		t.Error("a refusal lost its FailedPrecondition code")
	}

	for agent, n := range shiftsPerAgent(t, instances[0]) {
		if n != 1 {
			t.Errorf("operator %d has %d shifts, want 1", agent, n)
		}
	}
}
