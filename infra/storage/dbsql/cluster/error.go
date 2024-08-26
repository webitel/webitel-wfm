package cluster

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

func ParseError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return werror.NewDBNoRowsErr("dbsql.pg.query.no_rows")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return werror.NewDBUniqueViolationError("dbsql.pg.unique_violation", findColumn(pgErr.Detail), findValue(pgErr.Detail))
		case pgerrcode.ForeignKeyViolation:
			return werror.NewDBForeignKeyViolationError("dbsql.pg.foreign_key_violation", pgErr.ColumnName, findValue(pgErr.Detail), findForeignKeyTable(pgErr.Detail))
		case pgerrcode.CheckViolation:
			return werror.NewDBCheckViolationError("dbsql.pg.check_violation", pgErr.ConstraintName)
		case pgerrcode.NotNullViolation:
			return werror.NewDBNotNullViolationError("dbsql.pg.not_null_violation", pgErr.ColumnName)
		}
	}

	return werror.NewDBInternalError("dbsql.pg.query", err)
}

var columnFinder = regexp.MustCompile(`Key \((.+)\)=`)

// findColumn finds the column in the given pq Detail error string. If the
// column does not exist, the empty string is returned.
// Detail can look like this:
//
//	Key (id)=(3c7d2b4a-3fc8-4782-a518-4ce9efef51e7) already exists.
func findColumn(detail string) string {
	results := columnFinder.FindStringSubmatch(detail)
	if len(results) < 2 {
		return ""
	} else {
		return results[1]
	}
}

var valueFinder = regexp.MustCompile(`Key \(.+\)=\((.+)\)`)

// findColumn finds the column in the given pq Detail error string.
// If the column does not exist, the empty string is returned.
// Detail can look like this:
//
//	Key (id)=(3c7d2b4a-3fc8-4782-a518-4ce9efef51e7) already exists.
func findValue(detail string) string {
	results := valueFinder.FindStringSubmatch(detail)
	if len(results) < 2 {
		return ""
	}

	return results[1]
}

var foreignKeyFinder = regexp.MustCompile(`not present in table "(.+)"`)

// findForeignKeyTable finds the referenced table in the given pq Detail error
// string. If we can't find the table, we return the empty string.
// Detail can look like this:
//
//	Key (account_id)=(91f47e99-d616-4d8c-9c02-cbd13bceac60) is not present in table "accounts"
func findForeignKeyTable(detail string) string {
	results := foreignKeyFinder.FindStringSubmatch(detail)
	if len(results) < 2 {
		return ""
	}
	return results[1]
}

func formatPgError(err *pgconn.PgError) string {
	return fmt.Sprintf("%s: %s", err.Code, err.Message)
}
