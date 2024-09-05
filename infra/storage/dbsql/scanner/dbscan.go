package scanner

import (
	"github.com/georgysavva/scany/v2/dbscan"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/errors"
)

var _ Scanner = &DBScan{}

type DBScan struct {
	cli *dbscan.API
}

func NewDBScan() (*DBScan, error) {
	cli, err := dbscan.NewAPI()
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

func (f *DBScan) String() string {
	return "db-scan"
}

func (d *DBScan) ScanOne(dst interface{}, rows Rows) error {
	if err := d.cli.ScanOne(dst, rows); err != nil {
		return errors.ParseError(err)
	}

	return nil
}

func (d *DBScan) ScanAll(dst interface{}, rows Rows) error {
	if err := d.cli.ScanAll(dst, rows); err != nil {
		return errors.ParseError(err)
	}

	return nil
}
