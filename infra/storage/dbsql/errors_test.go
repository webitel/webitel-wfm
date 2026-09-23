package dbsql

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// A nil error must stay nil: every node method now hands ParseError the result
// of a call that usually succeeded.
func TestParseErrorNil(t *testing.T) {
	if err := ParseError(nil); err != nil {
		t.Fatalf("ParseError(nil) = %v, want nil", err)
	}
}

func TestParseError(t *testing.T) {
	tests := map[string]struct {
		in       error
		sentinel error
		code     codes.Code
	}{
		"no rows": {
			in:       fmt.Errorf("query: %w", pgx.ErrNoRows),
			sentinel: ErrNoRows,
			code:     codes.NotFound,
		},
		"unique violation": {
			in:       &pgconn.PgError{Code: pgerrcode.UniqueViolation, Detail: "Key (id)=(7) already exists."},
			sentinel: ErrUniqueViolation,
			code:     codes.Aborted,
		},
		"foreign key violation": {
			in:       &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation, Detail: `Key (team_id)=(3) is not present in table "teams"`},
			sentinel: ErrForeignKeyViolation,
			code:     codes.Aborted,
		},
		"not null violation": {
			in:       &pgconn.PgError{Code: pgerrcode.NotNullViolation, TableName: "working_schedule", ColumnName: "name"},
			sentinel: ErrNotNullViolation,
			code:     codes.Aborted,
		},
		"unmapped driver error": {
			in:       &pgconn.PgError{Code: pgerrcode.SyntaxError},
			sentinel: ErrInternal,
			code:     codes.Internal,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := ParseError(tt.in)
			if !errors.Is(got, tt.sentinel) {
				t.Errorf("ParseError(%v) is not %v", tt.in, tt.sentinel)
			}

			if errors.Code(got) != tt.code {
				t.Errorf("code = %v, want %v", errors.Code(got), tt.code)
			}

			// The driver error must stay reachable: storages branch on it.
			var pgErr *pgconn.PgError
			if _, isPg := tt.in.(*pgconn.PgError); isPg && !errors.As(got, &pgErr) {
				t.Error("the wrapped error no longer unwraps to *pgconn.PgError, so the SQLSTATE is lost")
			}
		})
	}
}

// A registered constraint contributes its own message; an unregistered one must
// not leave the sentinel with a dangling separator.
func TestParseErrorCheckViolationMessage(t *testing.T) {
	RegisterConstraint("test_check", "start must be before end")

	known := ParseError(&pgconn.PgError{Code: pgerrcode.CheckViolation, ConstraintName: "test_check"})
	if !strings.Contains(known.Error(), "start must be before end") {
		t.Errorf("registered constraint message missing: %q", known.Error())
	}

	unknown := ParseError(&pgconn.PgError{Code: pgerrcode.CheckViolation, ConstraintName: "not_registered"})
	if got := unknown.Error(); strings.HasSuffix(strings.TrimSpace(got), ":") {
		t.Errorf("unregistered constraint produced a dangling separator: %q", got)
	}
}
