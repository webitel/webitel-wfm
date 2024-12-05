package pg

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// rowsAdapter makes pgx.Rows compliant with the scanner.Rows interface.
// See dbsql.Rows for details.
type rowsAdapter struct {
	rows pgx.Rows
}

// newRowsAdapter returns a new rowsAdapter instance.
func newRowsAdapter(rows pgx.Rows) *rowsAdapter {
	return &rowsAdapter{rows: rows}
}

// Columns implements the dbscan.Rows.Columns method.
func (ra rowsAdapter) Columns() ([]string, error) {
	columns := make([]string, len(ra.rows.FieldDescriptions()))
	for i, fd := range ra.rows.FieldDescriptions() {
		columns[i] = fd.Name
	}

	return columns, nil
}

func (ra rowsAdapter) Types() ([]string, error) {
	typeMap := &pgtype.Map{}
	types := make([]string, len(ra.rows.FieldDescriptions()))
	for i, fd := range ra.rows.FieldDescriptions() {
		t, ok := typeMap.TypeForOID(fd.DataTypeOID)
		if !ok {
			return nil, fmt.Errorf("invalid type for OID: %d", fd.DataTypeOID)
		}

		switch fd.DataTypeOID {
		case pgtype.Int2OID, pgtype.Int4OID, pgtype.Int8OID, pgtype.Float4OID, pgtype.Float8OID, pgtype.NumericOID:
			types[i] = "number"
		case pgtype.TextOID, pgtype.QCharOID, pgtype.NameOID, pgtype.JSONOID, pgtype.JSONBOID:
			types[i] = "string"
		case pgtype.TimetzOID, pgtype.TimestamptzOID, pgtype.TimestampOID, pgtype.TimeOID, pgtype.DateOID:
			types[i] = "date"
		default:
			types[i] = t.Name
		}
	}

	return types, nil
}

// Close implements the dbscan.Rows.Close method.
func (ra rowsAdapter) Close() error {
	ra.rows.Close()

	return nil
}

// NextResultSet is currently always returning false.
func (ra rowsAdapter) NextResultSet() bool {
	// TODO: when pgx issue #308 and #1512 and is fixed maybe we can do something here.
	return false
}

func (ra rowsAdapter) Err() error {
	return ra.rows.Err()
}

func (ra rowsAdapter) Next() bool {
	return ra.rows.Next()
}

func (ra rowsAdapter) Scan(dest ...any) error {
	return ra.rows.Scan(dest...)
}

func (ra rowsAdapter) Values() ([]any, error) {
	return ra.rows.Values()
}
