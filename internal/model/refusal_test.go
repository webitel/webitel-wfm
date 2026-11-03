package model

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

var (
	v7 = Violation{OperatorID: 7, Date: pgtype.Date{Time: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), Valid: true}, Reason: "model.test.overlap"}
	v9 = Violation{OperatorID: 9, Date: NewDate(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC).Unix()), Reason: "model.test.day_locked"}

	errRefused = NewRefusal("model.test.refused", "test refusal")
)

func TestRefusal(t *testing.T) {
	refused := Refuse(errRefused, v7, v9)

	tests := map[string]error{
		"as is":     refused,
		"rollback":  errors.Wrap(refused, errors.WithValue("rollback", "conn closed")),
		"%w":        fmt.Errorf("service: %w", refused),
		"both wrap": fmt.Errorf("service: %w", errors.Wrap(refused, errors.WithValue("rollback", "conn closed"))),
	}
	for name, err := range tests {
		t.Run(name, func(t *testing.T) {
			if !errors.Is(err, errRefused) || errors.Code(err) != codes.FailedPrecondition || errors.ID(err) != "model.test.refused" {
				t.Errorf("Is, code, id = %v, %v, %q", errors.Is(err, errRefused), errors.Code(err), errors.ID(err))
			}

			if got := ViolationsOf(err); !reflect.DeepEqual(got, Violations{v7, v9}) {
				t.Errorf("ViolationsOf = %v, want [v7 v9]", got)
			}
		})
	}

	if Refuse(errRefused) != errRefused {
		t.Error("Refuse without violations must return the sentinel itself")
	}

	if got := ViolationsOf(errRefused); got != nil {
		t.Errorf("ViolationsOf(sentinel) = %v, want nil: Refuse mutated it", got)
	}
}

func TestViolationsString(t *testing.T) {
	want := "operator 7 on 2026-11-02: model.test.overlap, operator 9 on 2026-11-03: model.test.day_locked"
	if got := (Violations{v7, v9}).String(); got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}
