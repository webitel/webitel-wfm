package scanner

import (
	"reflect"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

var _ Scanner = &BatchScan{}

type BatchScan struct {
	cli *DBScan
}

func NewBatchScan() (*BatchScan, error) {
	v, err := NewDBScan()
	if err != nil {
		return nil, err
	}

	return &BatchScan{cli: v}, nil
}

func MustNewBatchScan() *BatchScan {
	v, err := NewBatchScan()
	if err != nil {
		panic(err)
	}

	return v
}

func (b *BatchScan) String() string {
	return "batch-scan"
}

func (b *BatchScan) ScanOne(dst interface{}, rows Rows) error {
	// TODO implement me
	panic("implement me")
}

func (b *BatchScan) ScanAll(dst interface{}, rows Rows) error {
	destSlice := reflect.ValueOf(dst)
	if destSlice.Kind() != reflect.Ptr {
		return werror.New("recieved non-pointer", werror.WithID("dbsql.cluster.batch"),
			werror.WithValue("type", destSlice.Type().String()),
		)
	}

	// Get the value that the pointer v points to.
	v := destSlice.Elem()
	if v.Kind() != reflect.Slice {
		return werror.New("can't fill non-slice value", werror.WithID("dbsql.cluster.batch"))
	}

	// Create a slice of dest type and set it to newly created slice
	// so we can merge it later.
	slv := reflect.MakeSlice(v.Type(), 0, 0)
	slv = reflect.AppendSlice(slv, destSlice.Elem())

	if err := b.cli.ScanAll(dst, rows); err != nil {
		return err
	}

	slv = reflect.AppendSlice(slv, destSlice.Elem())

	// Replace dest with merged slice.
	destSlice.Elem().Set(slv.Slice(0, slv.Len()))

	return nil
}
