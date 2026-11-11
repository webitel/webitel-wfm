package storage

import (
	"context"
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/internal/model"
)

// No database here: validation must run before the transaction.
func TestCreateShiftsValidatesBeforeTheTransaction(t *testing.T) {
	in := []*model.Shift{shiftAt(t, 1, "2026-11-02 08:00+02", "2026-11-02 08:00+02")}

	if _, err := (&AgentWorkingSchedule{}).CreateShifts(context.Background(), &model.SignedInUser{ID: 1, DomainID: 1}, 1, in); !errors.Is(err, model.ErrShiftDuration) {
		t.Errorf("err = %v, want %v", err, model.ErrShiftDuration)
	}
}
