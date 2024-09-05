package scanner

import (
	"fmt"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/errors"
	"github.com/webitel/webitel-wfm/internal/model"
)

var _ Scanner = &FrameScan{}

type FrameScan struct {
	cli *DBScan
}

func NewFrameScan() (*FrameScan, error) {
	v, err := NewDBScan()
	if err != nil {
		return nil, err
	}

	return &FrameScan{cli: v}, nil
}

func MustNewFrameScan() *FrameScan {
	v, err := NewFrameScan()
	if err != nil {
		panic(err)
	}

	return v
}

func (f *FrameScan) String() string {
	return "frame-scan"
}

func (f *FrameScan) ScanOne(dst any, rows Rows) error {
	return f.cli.ScanOne(dst, rows)
}

func (f *FrameScan) ScanAll(dst any, rows Rows) error {
	out, ok := dst.(*[]*model.ForecastCalculationResult)
	if !ok {
		panic(fmt.Errorf("frame scan: expected dst to be a []*ForecastCalculationResult, got %T", dst))
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	types, err := rows.Types()
	if err != nil {
		return err
	}

	items := make([]*model.ForecastCalculationResult, 0, len(columns))
	for i, col := range columns {
		field := &model.ForecastCalculationResult{
			Name:   col,
			Type:   types[i],
			Values: make([]any, 0),
		}

		items = append(items, field)
	}

	for {
		// first iterate over rows may be nop if not switched result set to next
		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				return errors.ParseError(err)
			}

			for i, v := range values {
				items[i].Values = append(items[i].Values, v)
			}
		}

		if !rows.NextResultSet() {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return errors.ParseError(err)
	}

	*out = items

	return nil
}
