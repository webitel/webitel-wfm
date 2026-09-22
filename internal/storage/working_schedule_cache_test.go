package storage

import (
	"context"
	"testing"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
)

type probeNode struct {
	dbsql.Node

	rows *[]*model.WorkingSchedule
}

func (p probeNode) Select(_ context.Context, dest any, _ string, _ ...any) error {
	out := dest.(*[]*model.WorkingSchedule)
	*out = *p.rows

	return nil
}

type probeStore struct {
	cluster.Store

	node probeNode
}

func (p probeStore) StandbyPreferred() dbsql.Node { return p.node }

func TestSearchWorkingScheduleCollectionCache(t *testing.T) {
	m, err := cache.New(&config.Cache{Size: 1024})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Stop()

	var rows []*model.WorkingSchedule

	scope := cache.NewScope[model.WorkingSchedule](m, "wfm.working_schedule")
	ws := &WorkingSchedule{db: probeStore{node: probeNode{rows: &rows}}, cache: scope}
	ctx := grpccontext.SetUser(context.Background(), &model.SignedInUser{Id: 1, DomainId: 1})

	// Cold cache, as after a restart. working_schedule_id carries no validate
	// rule, so a zero reaches storage and matches no row.
	rows = nil

	s1, _ := options.NewSearch(ctx, options.WithID(0))
	if _, err := ws.SearchWorkingSchedule(ctx, s1); err != nil {
		t.Fatal(err)
	}

	got, ok := scope.Key(1, 0).GetMany(ctx)
	t.Logf("collection key after read by id 0: ok=%v len=%d", ok, len(got))

	rows = []*model.WorkingSchedule{
		{DomainRecord: model.DomainRecord{Id: 7}, Name: "seven"},
		{DomainRecord: model.DomainRecord{Id: 8}, Name: "eight"},
	}

	s2, _ := options.NewSearch(ctx)

	out, err := ws.SearchWorkingSchedule(ctx, s2)
	if err != nil {
		t.Fatal(err)
	}

	if len(out) != 2 {
		t.Fatalf("search returned %d schedules, want 2: the read by id overwrote the collection key", len(out))
	}
}
