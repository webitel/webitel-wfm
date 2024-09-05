package pg

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RowsAdapter makes pgx.Rows compliant with the dbsql.Rows interface.
// See dbsql.Rows for details.
type RowsAdapter struct {
	pgx.Rows
}

// NewRowsAdapter returns a new rowsAdapter instance.
func NewRowsAdapter(rows pgx.Rows) *RowsAdapter {
	return &RowsAdapter{Rows: rows}
}

// Columns implements the dbscan.Rows.Columns method.
func (ra RowsAdapter) Columns() ([]string, error) {
	columns := make([]string, len(ra.Rows.FieldDescriptions()))
	for i, fd := range ra.Rows.FieldDescriptions() {
		columns[i] = fd.Name
	}

	return columns, nil
}

func (ra RowsAdapter) Types() ([]string, error) {
	typeMap := &pgtype.Map{}
	types := make([]string, len(ra.Rows.FieldDescriptions()))
	for i, fd := range ra.Rows.FieldDescriptions() {
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
func (ra RowsAdapter) Close() error {
	ra.Rows.Close()

	return nil
}

// NextResultSet is currently always returning false.
func (ra RowsAdapter) NextResultSet() bool {
	// TODO: when pgx issue #308 and #1512 and is fixed maybe we can do something here.
	return false
}
