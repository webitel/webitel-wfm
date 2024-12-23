package dbsql

import (
	"context"
	"reflect"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type Batcher interface {
	// Queue queues a query to batch.
	// Query can be an SQL query or the name of a prepared statement.
	Queue(query string, arguments ...any)

	Send(ctx context.Context) BatcherResults

	// Len returns number of queries that have been queued so far.
	Len() int
}

type BatcherResults interface {
	// Query reads the results from the next query in the batch
	// as if the query has been sent with Conn.Query.
	Query() (scanner.Rows, error)

	// Exec reads the results from the next query in the batch
	// as if the query has been sent with Conn.Exec.
	Exec() error

	// Close closes the batch operation, must be called before the underlying connection
	// can be used again. Any error that occurred during a batch operation may have made
	// it impossible to resyncronize the connection with the server.
	// In this case the underlying connection will have been closed.
	// Close is safe to call multiple times. If it returns an error subsequent calls
	// will return the same error. Callback functions will not be rerun
	Close() error
}

type sqlNodeBatch struct {
	batcher Batcher
	scanner scanner.Scanner
}

func newSqlNodeBatch(batcher Batcher) *sqlNodeBatch {
	return &sqlNodeBatch{
		batcher: batcher,
		scanner: scanner.MustNewBatchScan(),
	}
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
