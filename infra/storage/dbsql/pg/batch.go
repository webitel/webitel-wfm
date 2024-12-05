package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/batch"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
)

type batchAdapter struct {
	cli   *pgxpool.Pool
	batch *pgx.Batch
}

func newBatch(db *pgxpool.Pool) *batchAdapter {
	return &batchAdapter{
		cli:   db,
		batch: &pgx.Batch{},
	}
}

// Queue queues a query to batch.
// Query can be an SQL query or the name of a prepared statement.
func (b *batchAdapter) Queue(query string, arguments ...any) {
	b.batch.Queue(query, arguments...)
}

func (b *batchAdapter) Send(ctx context.Context) batch.BatcherResults {
	return newBatchResults(b.cli.SendBatch(ctx, b.batch))
}

// Len returns number of queries that have been queued so far.
func (b *batchAdapter) Len() int {
	return b.batch.Len()
}

type batchResults struct {
	res pgx.BatchResults
}

func newBatchResults(b pgx.BatchResults) *batchResults {
	return &batchResults{res: b}
}

// Query reads the results from the next query in the batch
// as if the query has been sent with Conn.Query.
// Prefer calling Query on the QueuedQuery.
func (b *batchResults) Query() (scanner.Rows, error) {
	rows, err := b.res.Query()
	if err != nil {
		return nil, err
	}

	return newRowsAdapter(rows), nil
}

// Exec reads the results from the next query in the batch
// as if the query has been sent with Conn.Exec.
// Prefer calling Exec on the QueuedQuery.
func (b *batchResults) Exec() error {
	_, err := b.res.Exec()
	if err != nil {
		return err
	}

	return nil
}

// Close closes the batch operation, must be called before the underlying connection
// can be used again. Any error that occurred during a batch operation may have made
// it impossible to resyncronize the connection with the server.
// In this case the underlying connection will have been closed.
// Close is safe to call multiple times. If it returns an error subsequent calls
// will return the same error. Callback functions will not be rerun
func (b *batchResults) Close() error {
	return b.res.Close()
}
