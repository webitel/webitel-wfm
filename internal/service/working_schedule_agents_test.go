package service

import (
	"context"
	"slices"
	"testing"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
)

type agentsStore struct {
	storage.WorkingScheduleManager

	added []int64
}

func (s *agentsStore) DeactivatedAgents(context.Context, int64, []int64) ([]int64, error) {
	return nil, nil
}

func (s *agentsStore) UpdateWorkingScheduleAddAgents(_ context.Context, _ *options.Read, agentIDs []int64) ([]*model.LookupItem, error) {
	s.added = agentIDs

	return nil, nil
}

func TestAddAgentsMatchesEachAgentOnce(t *testing.T) {
	ctx, _ := signedIn()
	store := &agentsStore{}
	svc := NewWorkingSchedule(store, newEngine(t, &fakeEngine{agents: []int64{3, 7}}), nil)

	read, err := options.NewRead(ctx, options.WithID(1))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.UpdateWorkingScheduleAddAgents(ctx, read, []int64{7, 3, 7}); err != nil {
		t.Fatalf("add a repeated agent: %v", err)
	}

	if want := []int64{3, 7}; !slices.Equal(store.added, want) {
		t.Errorf("stored %v, want %v", store.added, want)
	}
}
