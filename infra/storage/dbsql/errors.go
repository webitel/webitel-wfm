package dbsql

import (
	"errors"
	"regexp"
	"sync"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

var (
	ErrInternal = werror.Internal("internal server error", werror.WithID("dbsql.internal"))

	ErrNoRows              = werror.NotFound("entity does not exists or you do not have enough permissions to perform the operation", werror.WithID("dbsql.query.no_rows"))
	ErrUniqueViolation     = werror.Aborted("invalid input: entity already exists", werror.WithID("dbsql.unique_violation"))
	ErrForeignKeyViolation = werror.Aborted("invalid input: violates foreign key constraint", werror.WithID("dbsql.foreign_key_violation"))
	ErrCheckViolation      = werror.Aborted("invalid input: violates check constraint", werror.WithID("dbsql.check_violation"))
	ErrNotNullViolation    = werror.Aborted("invalid input: violates not null constraint: column can not be null", werror.WithID("dbsql.not_null_violation"))
	ErrEntityConflict      = werror.Aborted("invalid input: found more then one requested entity", werror.WithID("dbsql.conflict"))
)

func ParseError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return werror.Wrap(ErrNoRows, werror.WithCause(err))
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return werror.Wrap(ErrUniqueViolation, werror.WithCause(err),
				werror.WithValue("entity", findColumn(pgErr.Detail)+" = "+findValue(pgErr.Detail)),
			)
		case pgerrcode.ForeignKeyViolation:
			msg := "value is still referenced by the parent table"
			if findForeignKeyTable(pgErr.Detail) != "" {
				msg = "value isn't present in the parent table"
			}

			return werror.Wrap(ErrForeignKeyViolation, werror.WithCause(err), werror.AppendMessage(msg),
				werror.WithValue("value", findColumn(pgErr.Detail)+" = "+findValue(pgErr.Detail)),
				werror.WithValue("foreign_table", findForeignKeyTable(pgErr.Detail)),
			)
		case pgerrcode.CheckViolation:
			return werror.Wrap(ErrCheckViolation, werror.WithCause(err),
				werror.AppendMessage(checkViolationErrorRegistry[pgErr.ConstraintName]),
				werror.WithValue("constraint", pgErr.ConstraintName),
			)
		case pgerrcode.NotNullViolation:
			return werror.Wrap(ErrNotNullViolation, werror.WithCause(err),
				werror.WithValue("column", pgErr.TableName+"."+pgErr.ColumnName),
			)
		}
	}

	return werror.Wrap(ErrInternal, werror.WithCause(err))
}

var checkViolationErrorRegistry = map[string]string{}
var constraintMu sync.RWMutex

// RegisterConstraint register custom database check constraint (like "CHECK
// balance > 0").
// Postgres doesn't define a very useful message for constraint
// failures (new row for relation "accounts" violates check constraint), so you
// can define your own.
//   - name - should be the name of the constraint in the database.
//   - message - your own custom error message
//
// Panics if you attempt to register two constraints with the same name.
func RegisterConstraint(name, message string) {
	constraintMu.Lock()
	defer constraintMu.Unlock()
	if _, dup := checkViolationErrorRegistry[name]; dup {
		panic("register constraint called twice for name " + name)
	}

	checkViolationErrorRegistry[name] = message
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
