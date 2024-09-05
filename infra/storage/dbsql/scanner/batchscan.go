package scanner

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

func (f *BatchScan) String() string {
	return "batch-scan"
}

func (b *BatchScan) ScanOne(dst interface{}, rows Rows) error {
	// TODO implement me
	panic("implement me")
}

func (b *BatchScan) ScanAll(dst interface{}, rows Rows) error {
	// TODO implement me
	panic("implement me")
}
