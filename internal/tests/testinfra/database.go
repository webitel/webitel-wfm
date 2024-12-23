package testinfra

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/mock"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	mockdbsql "github.com/webitel/webitel-wfm/gen/go/mocks/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/pg"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
)

type TestStorageCluster struct {
	dbmock  pgxmock.PgxPoolIface
	cluster cluster.Store
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

	cl, err := cluster.New(log, []dbsql.Node{dbsql.New("mock", conn, scanner.MustNewDBScan())})
	if err != nil {
		return nil, err
	}

	return &TestStorageCluster{
		dbmock:  db,
		cluster: cl,
		queryer: conn,
	}, nil
}

func (t *TestStorageCluster) Store() cluster.Store {
	return t.cluster
}

func (t *TestStorageCluster) Mock() pgxmock.PgxPoolIface {
	return t.dbmock
}

func mockDatabase(t *testing.T, db pgxmock.PgxPoolIface) (dbsql.Database, error) {
	conn := mockdbsql.NewMockDatabase(t)
	conn.EXPECT().Exec(mock.Anything, mock.AnythingOfType("string")).
		RunAndReturn(func(ctx context.Context, sql string, args ...any) (int64, error) {
			ra, err := db.Exec(ctx, sql, args...)
			if err != nil {
				return 0, err
			}

			return ra.RowsAffected(), nil
		}).Maybe()

	conn.EXPECT().Query(mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		RunAndReturn(func(ctx context.Context, sql string, args ...any) (scanner.Rows, error) {
			rows, err := db.Query(ctx, sql, args...)
			if err != nil {
				return nil, err
			}

			return pg.NewRowsAdapter(rows), nil
		}).Maybe()

	return conn, nil
}
