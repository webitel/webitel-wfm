package dbsql

import (
	"context"
	"fmt"
	"reflect"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"

	"github.com/webitel/webitel-go-kit/infra/pgw"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type batch struct {
	pool  *pgw.Pool
	batch pgx.Batch
}

func (b *batch) Queue(query string, args ...any) {
	b.batch.Queue(query, args...)
}

// Select runs the queued queries and appends every result set to dest.
func (b *batch) Select(ctx context.Context, dest any) error {
	conn, err := b.pool.Acquire(ctx)
	if err != nil {
		return ParseError(err)
	}
	defer conn.Release()

	res := conn.SendBatch(ctx, &b.batch)
	for range b.batch.Len() {
		rows, err := res.Query()
		if err != nil {
			_ = res.Close()

			return ParseError(err)
		}

		if err := scanAppend(dest, rows); err != nil {
			_ = res.Close()

			return ParseError(err)
		}
	}

	return ParseError(res.Close())
}

// scanAppend scans rows into a fresh slice of dest's type and appends it, so
// the results of several queued queries accumulate in dest.
func scanAppend(dest any, rows pgx.Rows) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Slice {
		// ScanAll would have closed them; nothing else will.
		rows.Close()

		return errors.New("batch destination must be a pointer to a slice", errors.WithID("dbsql.batch.destination"),
			errors.WithValue("type", fmt.Sprintf("%T", dest)),
		)
	}

	part := reflect.New(v.Elem().Type())
	if err := pgxscan.ScanAll(part.Interface(), rows); err != nil {
		return err
	}

	v.Elem().Set(reflect.AppendSlice(v.Elem(), part.Elem()))

	return nil
}
