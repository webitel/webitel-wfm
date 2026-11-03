package dbsql

import (
	"cmp"
	"context"
	"math"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

const (
	// ASCII "wfm" plus the kind; teams sort before operators.
	teamLockClass     int32 = 0x77666D01
	operatorLockClass int32 = 0x77666D02

	lockSQL = "SELECT pg_advisory_xact_lock($1, $2)"
)

var ErrLockKeyRange = errors.Internal("advisory lock id does not fit int4", errors.WithID("dbsql.lock.key_range"))

type LockKey struct {
	class int32
	id    int64
}

func TeamLock(id int64) LockKey     { return LockKey{class: teamLockClass, id: id} }
func OperatorLock(id int64) LockKey { return LockKey{class: operatorLockClass, id: id} }

// The order is decided in Go: SQL gives no guarantee about the evaluation order of a volatile function.
func lockOrder(keys []LockKey) ([]LockKey, error) {
	sorted := slices.Clone(keys)
	for _, k := range sorted {
		if k.id < math.MinInt32 || k.id > math.MaxInt32 {
			return nil, errors.Wrap(ErrLockKeyRange, errors.WithValue("id", k.id))
		}
	}

	slices.SortFunc(sorted, func(a, b LockKey) int {
		return cmp.Or(cmp.Compare(a.class, b.class), cmp.Compare(a.id, b.id))
	})

	return slices.Compact(sorted), nil
}

// One batch: one round trip for every key, run in lockOrder's order.
func (s txStore) Lock(ctx context.Context, keys ...LockKey) error {
	sorted, err := lockOrder(keys)
	if err != nil || len(sorted) == 0 {
		return err
	}

	b := &pgx.Batch{}
	for _, k := range sorted {
		b.Queue(lockSQL, k.class, int32(k.id))
	}

	return ParseError(s.tx.SendBatch(ctx, b).Close())
}
