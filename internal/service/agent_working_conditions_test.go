package service

import (
	"testing"

	"github.com/stretchr/testify/mock"

	storagemocks "github.com/webitel/webitel-wfm/gen/go/mocks/internal_/storage"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
)

// The update checks the agent exists before it writes. The row it answers with
// comes from the write, so engine is not asked a second time.
func TestUpdateAgentWorkingConditionsChecksTheAgentOnce(t *testing.T) {
	ctx, _ := signedIn()

	read, err := options.NewRead(ctx, options.WithID(3))
	if err != nil {
		t.Fatal(err)
	}

	want := &model.AgentWorkingConditions{WorkingCondition: model.LookupItem{ID: 5}}

	st := storagemocks.NewMockAgentWorkingConditionsManager(t)
	st.EXPECT().UpdateAgentWorkingConditions(mock.Anything, read, want).Return(want, nil)

	fake := &fakeEngine{}

	out, err := NewAgentWorkingConditions(st, newEngine(t, fake)).UpdateAgentWorkingConditions(ctx, read, want)
	if err != nil {
		t.Fatal(err)
	}

	if out != want {
		t.Errorf("update returned %+v, want %+v", out, want)
	}

	if got := fake.agentReads.Load(); got != 1 {
		t.Errorf("engine was asked for the agent %d times, want 1", got)
	}
}
