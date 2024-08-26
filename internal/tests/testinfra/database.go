package testinfra

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/mock"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	dbsqlmock "github.com/webitel/webitel-wfm/gen/go/mocks/cluster"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/pg"
)

type TestStorageCluster struct {
	dbmock  pgxmock.PgxPoolIface
	cluster cluster.Store
	queryer cluster.Queryer
}

func NewTestStorageCluster(t *testing.T, log *wlog.Logger) (*TestStorageCluster, error) {
	t.Helper()

	db, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		return nil, err
	}

	conn, err := mockQueryer(t, db)
	if err != nil {
		return nil, err
	}

	cl, err := cluster.NewCluster(log, map[string]cluster.Queryer{"mock": conn}, cluster.WithUpdate(false))
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

func (t *TestStorageCluster) Queryer() cluster.Queryer {
	return t.queryer
}

func (t *TestStorageCluster) Mock() pgxmock.PgxPoolIface {
	return t.dbmock
}

func mockQueryer(t *testing.T, db pgxmock.PgxPoolIface) (cluster.Queryer, error) {
	conn := dbsqlmock.NewMockQueryer(t)
	conn.EXPECT().Close().Return().Maybe()
	conn.EXPECT().Exec(mock.Anything, mock.AnythingOfType("string"), mock.MatchedBy(func(args []any) bool { return true })).
		RunAndReturn(func(ctx context.Context, sql string, args []any) error {
			if _, err := db.Exec(ctx, sql, args...); err != nil {
				return err
			}

			return nil
		}).Maybe()

	conn.EXPECT().Query(mock.Anything, mock.AnythingOfType("string"), mock.MatchedBy(func(args []any) bool { return true })).
		RunAndReturn(func(ctx context.Context, sql string, args []any) (cluster.Rows, error) {
			rows, err := db.Query(ctx, sql, args...)
			if err != nil {
				return nil, err
			}

			return pg.NewRowsAdapter(rows), nil
		}).Maybe()

	return conn, nil
}
