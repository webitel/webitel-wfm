package pg

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
)

type BatchResults struct {
	pgx.BatchResults
}

func NewBatchResultsAdapter(b pgx.BatchResults) *BatchResults {
	return &BatchResults{BatchResults: b}
}

func (b *BatchResults) Query(_ context.Context, _ string, _ []any) (cluster.Rows, error) {
	rows, err := b.BatchResults.Query()
	if err != nil {
		return nil, err
	}

	return NewRowsAdapter(rows), nil
}

func (b *BatchResults) Exec(_ context.Context, _ string, _ []any) error {
	_, err := b.BatchResults.Exec()
	if err != nil {
		return err
	}

	return nil
}

func (b *BatchResults) Close() error {
	return b.BatchResults.Close()
}
