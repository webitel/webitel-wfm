package cluster

import (
	"context"
	"database/sql"
)

type Database interface {
	Queryer

	SendBatch(ctx context.Context, queries []*Query) BatchResults
	Close()
	Stdlib() *sql.DB
}

type Batcher interface {
	Queue(query string, args []any)
	Exec(ctx context.Context) error
	Select(ctx context.Context, dest any) error
}

// Queryer is an abstract database interface that can execute queries.
// This is used to decouple from any particular database library.
type Queryer interface {
	Exec(ctx context.Context, query string, args []any) error
	Query(ctx context.Context, query string, args []any) (Rows, error)
}

// Rows is an abstract database rows that dbscan can iterate over and get the data from.
// This interface is used to decouple from any particular database library.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Columns() ([]string, error)
	Scan(dest ...interface{}) error
	NextResultSet() bool
	Values() ([]any, error)
}
