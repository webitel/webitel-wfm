package scanner

import (
	"fmt"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

var _ Scanner = &ForecastScan{}

type ForecastScan struct {
	cli *DBScan
}

func NewForecastScan() (*ForecastScan, error) {
	v, err := NewDBScan()
	if err != nil {
		return nil, err
	}

	return &ForecastScan{cli: v}, nil
}

func MustNewForecastScan() *ForecastScan {
	v, err := NewForecastScan()
	if err != nil {
		panic(err)
	}

	return v
}

func (f *ForecastScan) String() string {
	return "forecast-scan"
}

func (f *ForecastScan) ScanOne(dst interface{}, rows Rows) error {
	// For cluster's check
	if _, ok := dst.(*bool); ok {
		return f.cli.ScanOne(dst, rows)
	}

	_, ok := dst.(**model.ForecastCalculationResult)
	if !ok {
		panic(fmt.Errorf("%s: expected dst to be a **model.ForecastCalculationResult, got %T", f.String(), dst))
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	if len(columns) != 0 && len(columns) != 2 {
		return werror.NewForecastProcedureResultErr("dbsql.scanner.forecastscan.one", len(columns))
	}

	return f.cli.ScanOne(dst, rows)
}

func (f *ForecastScan) ScanAll(dst interface{}, rows Rows) error {
	_, ok := dst.(*[]*model.ForecastCalculationResult)
	if !ok {
		panic(fmt.Errorf("%s: expected dst to be a *[]*model.ForecastCalculationResult, got %T", f.String(), dst))
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	if len(columns) != 0 && len(columns) != 2 {
		return werror.NewForecastProcedureResultErr("dbsql.scanner.forecastscan.all", len(columns))
	}

	return f.cli.ScanAll(dst, rows)
}
