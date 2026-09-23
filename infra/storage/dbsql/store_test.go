package dbsql

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// errNode stands in for an unreachable role. Every method must fail rather than
// dereference a pool that was never handed out, and must keep the pool
// manager's reason as the cause.
func TestErrNode(t *testing.T) {
	cause := errors.New("no primary registered")
	n := errNode{err: cause}
	ctx := context.Background()

	calls := map[string]error{
		"select":       n.Select(ctx, nil, "SELECT 1"),
		"get":          n.Get(ctx, nil, "SELECT 1"),
		"exec":         n.Exec(ctx, "SELECT 1"),
		"batch select": n.Batch().Select(ctx, nil),
	}

	for name, err := range calls {
		if err == nil {
			t.Errorf("%s on a dead node returned no error", name)

			continue
		}

		if !errors.Is(err, errDatabaseNodeDead) {
			t.Errorf("%s error is not errDatabaseNodeDead: %v", name, err)
		}

		if errors.Code(err) != codes.Internal {
			t.Errorf("%s code = %v, want Internal", name, errors.Code(err))
		}
	}

	// Queue must stay a no-op so a caller can build a batch before discovering
	// the node is gone.
	n.Batch().Queue("SELECT 1")
}

// stubRows records whether the scanner closed it. Only Close is ever reached on
// the path under test.
type stubRows struct {
	pgx.Rows

	closed bool
}

func (r *stubRows) Close() { r.closed = true }

// A destination that is not a pointer to a slice is a programming error, but it
// must be reported rather than panic, and it must not leak the result set.
func TestScanAppendRejectsBadDestination(t *testing.T) {
	var notASlice int

	for name, dest := range map[string]any{
		"nil":                  nil,
		"non-pointer":          []int64{},
		"pointer to non-slice": &notASlice,
	} {
		t.Run(name, func(t *testing.T) {
			rows := &stubRows{}

			err := scanAppend(dest, rows)
			if err == nil {
				t.Fatalf("scanAppend(%T) returned no error", dest)
			}

			if !rows.closed {
				t.Error("the rejected result set was left open")
			}
		})
	}
}
