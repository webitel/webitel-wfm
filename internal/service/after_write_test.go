package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	storagemocks "github.com/webitel/webitel-wfm/gen/go/mocks/internal_/storage"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
)

// afterWrite matches a read of the given row that carries the after-write
// intent.
func afterWrite(id int64) any {
	return mock.MatchedBy(func(r *options.Read) bool { return r.ID() == id && r.AfterWrite() })
}

// A lookup's create and update answer with the row read back after the write.
// That read must carry the after-write intent, or storage may serve it from a
// lagging standby: NotFound right after a create, stale fields after an update.
func TestLookupWritesReadBackAfterWrite(t *testing.T) {
	ctx := grpccontext.SetUser(context.Background(), &model.SignedInUser{Id: 1, DomainId: 1})

	newRead := func(t *testing.T) *options.Read {
		t.Helper()

		read, err := options.NewRead(ctx, options.WithID(7))
		if err != nil {
			t.Fatal(err)
		}

		return read
	}

	lookups := map[string]struct {
		create func(t *testing.T) error
		update func(t *testing.T) error
	}{
		"pause template": {
			create: func(t *testing.T) error {
				t.Helper()

				st := storagemocks.NewMockPauseTemplateManager(t)
				st.EXPECT().CreatePauseTemplate(mock.Anything, mock.Anything, mock.Anything).Return(int64(7), nil)
				st.EXPECT().ReadPauseTemplate(mock.Anything, afterWrite(7)).Return(&model.PauseTemplate{}, nil)

				_, err := NewPauseTemplate(st).CreatePauseTemplate(ctx, newRead(t), &model.PauseTemplate{})

				return err
			},
			update: func(t *testing.T) error {
				t.Helper()

				st := storagemocks.NewMockPauseTemplateManager(t)
				st.EXPECT().UpdatePauseTemplate(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				st.EXPECT().ReadPauseTemplate(mock.Anything, afterWrite(7)).Return(&model.PauseTemplate{}, nil)

				_, err := NewPauseTemplate(st).UpdatePauseTemplate(ctx, newRead(t), &model.PauseTemplate{})

				return err
			},
		},
		"shift template": {
			create: func(t *testing.T) error {
				t.Helper()

				st := storagemocks.NewMockShiftTemplateManager(t)
				st.EXPECT().CreateShiftTemplate(mock.Anything, mock.Anything, mock.Anything).Return(int64(7), nil)
				st.EXPECT().ReadShiftTemplate(mock.Anything, afterWrite(7)).Return(&model.ShiftTemplate{}, nil)

				_, err := NewShiftTemplate(st).CreateShiftTemplate(ctx, newRead(t), &model.ShiftTemplate{})

				return err
			},
			update: func(t *testing.T) error {
				t.Helper()

				st := storagemocks.NewMockShiftTemplateManager(t)
				st.EXPECT().UpdateShiftTemplate(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				st.EXPECT().ReadShiftTemplate(mock.Anything, afterWrite(7)).Return(&model.ShiftTemplate{}, nil)

				_, err := NewShiftTemplate(st).UpdateShiftTemplate(ctx, newRead(t), &model.ShiftTemplate{})

				return err
			},
		},
		"working condition": {
			create: func(t *testing.T) error {
				t.Helper()

				st := storagemocks.NewMockWorkingConditionManager(t)
				st.EXPECT().CreateWorkingCondition(mock.Anything, mock.Anything, mock.Anything).Return(int64(7), nil)
				st.EXPECT().ReadWorkingCondition(mock.Anything, afterWrite(7)).Return(&model.WorkingCondition{}, nil)

				_, err := NewWorkingCondition(st).CreateWorkingCondition(ctx, newRead(t), &model.WorkingCondition{})

				return err
			},
			update: func(t *testing.T) error {
				t.Helper()

				st := storagemocks.NewMockWorkingConditionManager(t)
				st.EXPECT().UpdateWorkingCondition(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				st.EXPECT().ReadWorkingCondition(mock.Anything, afterWrite(7)).Return(&model.WorkingCondition{}, nil)

				_, err := NewWorkingCondition(st).UpdateWorkingCondition(ctx, newRead(t), &model.WorkingCondition{})

				return err
			},
		},
	}

	for name, tc := range lookups {
		t.Run(name, func(t *testing.T) {
			if err := tc.create(t); err != nil {
				t.Errorf("create: %v", err)
			}

			if err := tc.update(t); err != nil {
				t.Errorf("update: %v", err)
			}
		})
	}
}
