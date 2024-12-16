package dbsql

import (
	"context"
	"reflect"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/batch"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type BatchNode interface {
	Queue(query string, args ...any)
	Select(ctx context.Context, dest any) error
	Exec(ctx context.Context) error
}

type sqlNodeBatch struct {
	batcher batch.Batcher
	scanner scanner.Scanner
}

func newSqlNodeBatch(batcher batch.Batcher) *sqlNodeBatch {
	return &sqlNodeBatch{batcher: batcher, scanner: scanner.MustNewBatchScan()}
}

func (n *sqlNodeBatch) Queue(sql string, args ...any) {
	n.batcher.Queue(sql, args...)
}

func (n *sqlNodeBatch) Select(ctx context.Context, dest any) error {
	destSlice := reflect.ValueOf(dest)
	if destSlice.Kind() != reflect.Ptr {
		return werror.Wrap(ErrInternal, werror.WithID("dbsql.cluster.batch"),
			werror.WithCause(werror.New("recieved non-pointer", werror.WithValue("type", destSlice.Type().String()))),
		)
	}

	// Get the value that the pointer v points to.
	v := destSlice.Elem()
	if v.Kind() != reflect.Slice {
		return werror.Wrap(ErrInternal, werror.WithID("dbsql.cluster.batch"),
			werror.WithCause(werror.New("can't fill non-slice value")),
		)
	}

	// Create a slice of dest type and set it to newly created slice
	// so we can merge it later,
	v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	queryer := n.batcher.Send(ctx)
	for i := 0; i < n.batcher.Len(); i++ {
		rows, err := queryer.Query()
		if err != nil {
			return ParseError(err)
		}

		if err := n.scanner.ScanAll(dest, rows); err != nil {
			return ParseError(err)
		}

		v = reflect.AppendSlice(v, destSlice.Elem())
	}

	// Replace dest with merged slice.
	destSlice.Elem().Set(v.Slice(0, v.Len()))

	return queryer.Close()
}

func (n *sqlNodeBatch) Exec(ctx context.Context) error {
	queryer := n.batcher.Send(ctx)
	for i := 0; i < n.batcher.Len(); i++ {
		if err := queryer.Exec(); err != nil {
			return ParseError(err)
		}
	}

	return queryer.Close()
}
