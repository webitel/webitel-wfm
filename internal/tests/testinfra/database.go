package testinfra

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/mock"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/pg"
)

type TestStorageCluster struct {
	dbmock  pgxmock.PgxPoolIface
	cluster dbsql.Store
	queryer dbsql.Database
}

func NewTestStorageCluster(t *testing.T, log *wlog.Logger) (*TestStorageCluster, error) {
	t.Helper()

	db, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		return nil, err
	}

	conn, err := mockDatabase(t, db)
	if err != nil {
		return nil, err
	}

	cl, err := dbsql.NewCluster(log, map[string]dbsql.Database{"mock": conn})
	if err != nil {
		return nil, err
	}

	return &TestStorageCluster{
		dbmock:  db,
		cluster: cl,
		queryer: conn,
	}, nil
}

func (t *TestStorageCluster) Store() dbsql.Store {
	return t.cluster
}

func (t *TestStorageCluster) Queryer() dbsql.Queryer {
	return t.queryer
}

func (t *TestStorageCluster) Mock() pgxmock.PgxPoolIface {
	return t.dbmock
}

func mockDatabase(t *testing.T, db pgxmock.PgxPoolIface) (dbsql.Database, error) {
	conn := dbsqlmock.NewMockDatabase(t)
	conn.EXPECT().Exec(mock.Anything, mock.AnythingOfType("string")).
		RunAndReturn(func(ctx context.Context, sql string, args ...any) error {
			if _, err := db.Exec(ctx, sql, args...); err != nil {
				return err
			}

			return nil
		}).Maybe()

	conn.EXPECT().Query(mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		RunAndReturn(func(ctx context.Context, sql string, args ...any) (dbsql.Rows, error) {
			rows, err := db.Query(ctx, sql, args...)
			if err != nil {
				return nil, err
			}

			return pg.NewRowsAdapter(rows), nil
		}).Maybe()

	return conn, nil
}
