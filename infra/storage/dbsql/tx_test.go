package dbsql

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/infra/pgw"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type stubTx struct {
	pgx.Tx

	execErr, queryErr, commitErr, rollbackErr error

	execs, batches, savepoints int
	committed, rolledBack      bool
	rollbackCtxErr             error
	sent                       *pgx.Batch
}

func (t *stubTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	t.execs++

	return pgconn.CommandTag{}, t.execErr
}

func (t *stubTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, t.queryErr
}

func (t *stubTx) SendBatch(_ context.Context, b *pgx.Batch) pgx.BatchResults {
	t.batches++
	t.sent = b

	return stubBatchResults{}
}

func (t *stubTx) Begin(context.Context) (pgx.Tx, error) {
	t.savepoints++

	return t, nil
}

func (t *stubTx) Commit(context.Context) error {
	t.committed = true

	return t.commitErr
}

func (t *stubTx) Rollback(ctx context.Context) error {
	t.rolledBack = true
	t.rollbackCtxErr = ctx.Err()

	return t.rollbackErr
}

type stubBatchResults struct {
	pgx.BatchResults
}

func (stubBatchResults) Close() error { return nil }

type stubBeginner struct {
	tx   *stubTx
	err  error
	opts []pgx.TxOptions
}

func (b *stubBeginner) BeginTx(_ context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	b.opts = append(b.opts, opts)
	if b.err != nil {
		return nil, b.err
	}

	return b.tx, nil
}

var errRefused = errors.New("schedule is being planned", errors.WithCode(codes.FailedPrecondition))

func TestInTxCommits(t *testing.T) {
	tx := &stubTx{}
	db := &stubBeginner{tx: tx}

	if err := inTx(context.Background(), db, func(context.Context, Tx) error { return nil }); err != nil {
		t.Fatalf("inTx = %v, want nil", err)
	}

	if !tx.committed || tx.rolledBack {
		t.Errorf("committed = %v, rolled back = %v; want a commit only", tx.committed, tx.rolledBack)
	}

	if len(db.opts) != 1 || db.opts[0].IsoLevel != pgx.ReadCommitted {
		t.Errorf("began %+v, want one READ COMMITTED transaction", db.opts)
	}
}

func TestInTxRollsBackOnError(t *testing.T) {
	for name, rollbackErr := range map[string]error{
		"rollback succeeds": nil,
		"rollback fails":    errors.New("conn closed"),
	} {
		t.Run(name, func(t *testing.T) {
			tx := &stubTx{rollbackErr: rollbackErr}
			ctx, cancel := context.WithCancel(context.Background())

			err := inTx(ctx, &stubBeginner{tx: tx}, func(context.Context, Tx) error {
				cancel()

				return errRefused
			})

			if !errors.Is(err, errRefused) || errors.Code(err) != codes.FailedPrecondition {
				t.Errorf("inTx = %v (%v), want fn's error with its code", err, errors.Code(err))
			}

			if tx.committed || !tx.rolledBack {
				t.Errorf("committed = %v, rolled back = %v; want a rollback only", tx.committed, tx.rolledBack)
			}

			if tx.rollbackCtxErr != nil {
				t.Errorf("rollback ran on a canceled ctx: %v", tx.rollbackCtxErr)
			}

			got, ok := errors.Lookup(err, "rollback")
			if want := rollbackErr != nil; ok != want || (want && got != rollbackErr.Error()) {
				t.Errorf("rollback annotation = %v (%v), want present = %v", got, ok, want)
			}
		})
	}
}

func TestInTxRollsBackOnPanic(t *testing.T) {
	tx := &stubTx{}
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		if p := recover(); p != "boom" {
			t.Errorf("recovered %v, want the panic re-raised", p)
		}

		if tx.committed || !tx.rolledBack {
			t.Errorf("committed = %v, rolled back = %v; want a rollback only", tx.committed, tx.rolledBack)
		}

		if tx.rollbackCtxErr != nil {
			t.Errorf("rollback ran on a canceled ctx: %v", tx.rollbackCtxErr)
		}
	}()

	_ = inTx(ctx, &stubBeginner{tx: tx}, func(context.Context, Tx) error {
		cancel()
		panic("boom")
	})
}

func TestInTxRollsBackWhenCtxEndsBeforeCommit(t *testing.T) {
	tx := &stubTx{}
	ctx, cancel := context.WithCancel(context.Background())

	err := inTx(ctx, &stubBeginner{tx: tx}, func(context.Context, Tx) error {
		cancel()

		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("inTx = %v, want context.Canceled as the cause", err)
	}

	if tx.committed || !tx.rolledBack || tx.rollbackCtxErr != nil {
		t.Errorf("committed = %v, rolled back = %v (ctx err %v); want a rollback on a live ctx",
			tx.committed, tx.rolledBack, tx.rollbackCtxErr)
	}
}

func TestInTxJoins(t *testing.T) {
	tx := &stubTx{}
	db := &stubBeginner{tx: tx}
	ctx := context.Background()

	err := inTx(ctx, db, func(ctx context.Context, outer Tx) error {
		return outer.InTx(ctx, func(ctx context.Context, inner Tx) error {
			if err := inner.Primary().Exec(ctx, "SELECT 1"); err != nil {
				return err
			}

			if err := inner.StandbyPreferred().Exec(ctx, "SELECT 1"); err != nil {
				return err
			}

			var dest []int64

			return inner.Primary().Batch().Select(ctx, &dest)
		})
	})
	if err != nil {
		t.Fatalf("inTx = %v, want nil", err)
	}

	if len(db.opts) != 1 || tx.savepoints != 0 {
		t.Errorf("began %d transactions and %d savepoints, want 1 and 0", len(db.opts), tx.savepoints)
	}

	if tx.execs != 2 || tx.batches != 1 {
		t.Errorf("tx got %d execs and %d batches, want 2 and 1", tx.execs, tx.batches)
	}

	if !tx.committed {
		t.Error("the joined transaction was not committed")
	}
}

func TestInTxParsesDriverErrors(t *testing.T) {
	unique := &pgconn.PgError{Code: pgerrcode.UniqueViolation}

	tests := map[string]struct {
		db       *stubBeginner
		fn       func(context.Context, Tx) error
		sentinel error
	}{
		"begin": {
			db:       &stubBeginner{err: errors.New("conn busy")},
			sentinel: ErrInternal,
		},
		"exec": {
			db:       &stubBeginner{tx: &stubTx{execErr: unique}},
			fn:       func(ctx context.Context, tx Tx) error { return tx.Primary().Exec(ctx, "INSERT") },
			sentinel: ErrUniqueViolation,
		},
		"get": {
			db: &stubBeginner{tx: &stubTx{queryErr: unique}},
			fn: func(ctx context.Context, tx Tx) error {
				var id int64

				return tx.Primary().Get(ctx, &id, "INSERT … RETURNING id")
			},
			sentinel: ErrUniqueViolation,
		},
		"select": {
			db: &stubBeginner{tx: &stubTx{queryErr: unique}},
			fn: func(ctx context.Context, tx Tx) error {
				var ids []int64

				return tx.Primary().Select(ctx, &ids, "INSERT … RETURNING id")
			},
			sentinel: ErrUniqueViolation,
		},
		"commit": {
			db:       &stubBeginner{tx: &stubTx{commitErr: unique}},
			sentinel: ErrUniqueViolation,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fn := tc.fn
			if fn == nil {
				fn = func(context.Context, Tx) error { return nil }
			}

			if err := inTx(context.Background(), tc.db, fn); !errors.Is(err, tc.sentinel) {
				t.Errorf("inTx = %v, want %v", err, tc.sentinel)
			}
		})
	}
}

func TestDBInTxOnDeadPrimary(t *testing.T) {
	called := false

	err := New(&pgw.PoolManager{}).InTx(context.Background(), func(context.Context, Tx) error {
		called = true

		return nil
	})

	if !errors.Is(err, errDatabaseNodeDead) || !errors.Is(err, pgw.ErrUnreachable) {
		t.Errorf("InTx = %v, want errDatabaseNodeDead caused by pgw.ErrUnreachable", err)
	}

	if called {
		t.Error("fn ran without a primary")
	}
}

func TestLockTakesKeysInOneOrder(t *testing.T) {
	stub := &stubTx{}
	keys := []LockKey{OperatorLock(5), TeamLock(9), OperatorLock(2), OperatorLock(5), TeamLock(1)}

	if err := (txStore{tx: stub}).Lock(context.Background(), keys...); err != nil {
		t.Fatalf("Lock = %v, want nil", err)
	}

	if stub.batches != 1 {
		t.Fatalf("batches = %d, want 1", stub.batches)
	}

	want := [][]any{
		{teamLockClass, int64(1)}, {teamLockClass, int64(9)}, {operatorLockClass, int64(2)}, {operatorLockClass, int64(5)},
	}
	if len(stub.sent.QueuedQueries) != len(want) {
		t.Fatalf("queued %d locks, want %d", len(stub.sent.QueuedQueries), len(want))
	}

	for i, q := range stub.sent.QueuedQueries {
		if q.SQL != lockSQL || !reflect.DeepEqual(q.Arguments, want[i]) {
			t.Errorf("lock %d = %q %v, want %q %v", i, q.SQL, q.Arguments, lockSQL, want[i])
		}
	}

	if keys[0] != OperatorLock(5) || keys[4] != TeamLock(1) {
		t.Errorf("Lock reordered the caller's keys: %v", keys)
	}
}

func TestLockRejectsIDsOutsideInt4(t *testing.T) {
	tests := map[string]struct {
		keys    []LockKey
		batches int
		wantErr bool
	}{
		"operator above int4":         {keys: []LockKey{OperatorLock(math.MaxInt32 + 1)}, wantErr: true},
		"team below int4":             {keys: []LockKey{TeamLock(math.MinInt32 - 1)}, wantErr: true},
		"one bad key among good ones": {keys: []LockKey{OperatorLock(1), OperatorLock(math.MaxInt32 + 1)}, wantErr: true},
		"int4 boundaries":             {keys: []LockKey{OperatorLock(math.MaxInt32), TeamLock(math.MinInt32)}, batches: 1},
		"no keys":                     {},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stub := &stubTx{}

			err := (txStore{tx: stub}).Lock(context.Background(), tt.keys...)
			if tt.wantErr && (!errors.Is(err, ErrLockKeyRange) || errors.ID(err) != "dbsql.lock.key_range") {
				t.Errorf("Lock = %v, want ErrLockKeyRange", err)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Lock = %v, want nil", err)
			}

			if stub.batches != tt.batches {
				t.Errorf("batches = %d, want %d", stub.batches, tt.batches)
			}
		})
	}
}

func TestRootStoreCannotLock(t *testing.T) {
	if _, ok := any((*DB)(nil)).(Tx); ok {
		t.Error("*DB implements Tx: Lock must be reachable only through InTx")
	}
}
