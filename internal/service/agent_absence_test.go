package service

import (
	"testing"

	"github.com/stretchr/testify/mock"

	storagemocks "github.com/webitel/webitel-wfm/gen/go/mocks/internal_/storage"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
)

// A bulk create answers with every agent it wrote. It is not a page of a list,
// so a batch the size of a page keeps its last agent.
func TestCreateAgentsAbsencesReturnsEveryAgent(t *testing.T) {
	ctx, _ := signedIn()

	search, err := options.NewSearch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	rows := make([]*model.AgentAbsences, options.DefaultSearchSize)
	for i := range rows {
		rows[i] = &model.AgentAbsences{Agent: model.LookupItem{ID: int64(i + 1)}}
	}

	st := storagemocks.NewMockAgentAbsenceManager(t)
	st.EXPECT().CreateAgentsAbsences(mock.Anything, search, mock.Anything).Return(rows, nil)

	out, err := NewAgentAbsence(st, nil, nil).CreateAgentsAbsences(ctx, search, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(out) != len(rows) {
		t.Errorf("create returned %d agents, want %d", len(out), len(rows))
	}
}
