package cluster

import (
	"context"
	"fmt"
	"reflect"

	"github.com/georgysavva/scany/v2/dbscan"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

type BatchResults interface {
	Query(_ context.Context, _ string, _ []any) (Rows, error)
	Exec(_ context.Context, _ string, _ []any) error
	Close() error
}

type Query struct {
	SQL  string
	Args []any
}

type sqlNodeBatch struct {
	db      Database
	scanner *dbscan.API
	queue   []*Query
}

func newSqlNodeBatch(db Database, scanner *dbscan.API) *sqlNodeBatch {
	return &sqlNodeBatch{db: db, scanner: scanner, queue: make([]*Query, 0)}
}

func (n *sqlNodeBatch) Queue(sql string, args []any) {
	n.queue = append(n.queue, &Query{SQL: sql, Args: args})
}

func (n *sqlNodeBatch) Select(ctx context.Context, dest any) error {
	destSlice := reflect.ValueOf(dest)
	if destSlice.Kind() != reflect.Ptr {
		return werror.NewDBInternalError("dbsql.cluster.batch", fmt.Errorf("recieved non-pointer %v", destSlice.Type()))
	}

	// Get the value that the pointer v points to.
	v := destSlice.Elem()
	if v.Kind() != reflect.Slice {
		return werror.NewDBInternalError("dbsql.cluster.batch", fmt.Errorf("can't fill non-slice value"))
	}

	// Create a slice of dest type and set it to newly created slice
	// so we can merge it later,
	v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	queryer := n.db.SendBatch(ctx, n.queue)
	for range n.queue {
		rows, err := queryer.Query(ctx, "", nil)
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
	queryer := n.db.SendBatch(ctx, n.queue)
	for range n.queue {
		if err := queryer.Exec(ctx, "", nil); err != nil {
			return ParseError(err)
		}
	}

	return queryer.Close()
}
