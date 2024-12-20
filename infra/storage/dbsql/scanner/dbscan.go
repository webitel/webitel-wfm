package scanner

import (
	"database/sql"

	"github.com/georgysavva/scany/v2/dbscan"
)

var _ Scanner = &DBScan{}

type DBScan struct {
	cli *dbscan.API
}

func NewDBScan() (*DBScan, error) {
	cli, err := dbscan.NewAPI(dbscan.WithScannableTypes((*sql.Scanner)(nil)))
	if err != nil {
		return nil, err
	}

	return &DBScan{cli: cli}, nil
}

func MustNewDBScan() *DBScan {
	cli, err := NewDBScan()
	if err != nil {
		panic(err)
	}

	return cli
}

func (d *DBScan) String() string {
	return "db-scan"
}

func (d *DBScan) ScanOne(dst interface{}, rows Rows) error {
	if err := d.cli.ScanOne(dst, rows); err != nil {
		return err
	}

	return nil
}

func (d *DBScan) ScanAll(dst interface{}, rows Rows) error {
	if err := d.cli.ScanAll(dst, rows); err != nil {
		return err
	}

	return nil
}
