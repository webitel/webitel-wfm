package scanner

// Rows is an abstract database rows that dbscan can iterate over and get the data from.
// This interface is used to decouple from any particular database library.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Columns() ([]string, error)
	Types() ([]string, error)
	Scan(dest ...any) error
	NextResultSet() bool
	Values() ([]any, error)
}

// Scanner used to configure dbscan API, scan batch results or
// frame with any values, returning its type.
//
// In case of unique / check / foreign key violations, the driver returns
// error only after rows.Close call (or while rows.Next returns false).
// That's why we should parse database error while scanning rows also.
type Scanner interface {
	String() string
	ScanOne(dst any, rows Rows) error
	ScanAll(dst any, rows Rows) error
}
