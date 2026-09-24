package storage

import (
	"context"
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/cache"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
)

// probeNode answers Select with canned rows and records that it was asked, so a
// test can tell which role of the cluster a query was routed to.
type probeNode struct {
	dbsql.Node

	role  string
	rows  *[]*model.WorkingSchedule
	id    int64
	asked *[]string
}

func (p probeNode) Select(_ context.Context, dest any, _ string, _ ...any) error {
	*p.asked = append(*p.asked, p.role)

	out, ok := dest.(*[]*model.WorkingSchedule)
	if !ok {
		return nil
	}

	*out = *p.rows

	return nil
}

func (p probeNode) Get(_ context.Context, dest any, _ string, _ ...any) error {
	*p.asked = append(*p.asked, p.role)

	if out, ok := dest.(*int64); ok {
		*out = p.id
	}

	return nil
}

func (p probeNode) Exec(_ context.Context, _ string, _ ...any) error {
	*p.asked = append(*p.asked, p.role)

	return nil
}

type probeStore struct {
	primary probeNode
	standby probeNode
}

func (p *probeStore) Primary() dbsql.Node          { return p.primary }
func (p *probeStore) StandbyPreferred() dbsql.Node { return p.standby }

// newProbe wires a WorkingSchedule onto canned rows. primaryRows and
// standbyRows are separate so a test can simulate replication lag.
func newProbe(t *testing.T, primaryRows, standbyRows *[]*model.WorkingSchedule, id int64) (*WorkingSchedule, *[]string) {
	t.Helper()

	asked := &[]string{}
	db := &probeStore{
		primary: probeNode{role: "primary", rows: primaryRows, id: id, asked: asked},
		standby: probeNode{role: "standby", rows: standbyRows, id: id, asked: asked},
	}

	return &WorkingSchedule{db: db, items: cache.Noop[string, model.WorkingSchedule]()}, asked
}

func userCtx(domain, user int64) context.Context {
	return grpccontext.SetUser(context.Background(), &model.SignedInUser{Id: user, DomainId: domain})
}

func schedule(id int64, name string) []*model.WorkingSchedule {
	return []*model.WorkingSchedule{{DomainRecord: model.DomainRecord{Id: id}, Name: name}}
}

// A row read straight after a write must come from the primary: a standby may
// not have replicated it yet, and the stale row would then be cached.
func TestUpdateWorkingScheduleReadsBackFromPrimary(t *testing.T) {
	primary := schedule(7, "renamed")
	standby := schedule(7, "stale")

	ws, asked := newProbe(t, &primary, &standby, 7)
	ctx := userCtx(1, 1)

	in := &model.WorkingSchedule{DomainRecord: model.DomainRecord{Id: 7}, Name: "renamed"}

	out, err := ws.UpdateWorkingSchedule(ctx, &model.SignedInUser{Id: 1, DomainId: 1}, in)
	if err != nil {
		t.Fatal(err)
	}

	if out.Name != "renamed" {
		t.Errorf("update returned %q, want %q: the read-back went to a lagging standby", out.Name, "renamed")
	}

	for _, role := range *asked {
		if role == "standby" {
			t.Fatalf("update touched the standby: %v", *asked)
		}
	}
}

func TestCreateWorkingScheduleReadsBackFromPrimary(t *testing.T) {
	primary := schedule(9, "created")

	var standby []*model.WorkingSchedule // not replicated yet

	ws, asked := newProbe(t, &primary, &standby, 9)
	ctx := userCtx(1, 1)

	in := &model.WorkingSchedule{Name: "created"}

	out, err := ws.CreateWorkingSchedule(ctx, &model.SignedInUser{Id: 1, DomainId: 1}, in)
	if err != nil {
		t.Fatalf("create of a committed row failed: %v", err)
	}

	if out.Name != "created" {
		t.Errorf("create returned %q, want %q", out.Name, "created")
	}

	for _, role := range *asked {
		if role == "standby" {
			t.Fatalf("create touched the standby: %v", *asked)
		}
	}
}

// A field mask makes the row partial, so it must neither be served from nor
// written to the entity cache.
func TestReadWorkingScheduleFieldMaskBypassesCache(t *testing.T) {
	rows := []*model.WorkingSchedule{{DomainRecord: model.DomainRecord{Id: 7}}}

	ws, _ := newProbe(t, &rows, &rows, 7)
	ctx := userCtx(1, 1)

	masked, err := options.NewRead(ctx, options.WithID(7), options.WithFields([]string{"id"}))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ws.ReadWorkingSchedule(ctx, masked); err != nil {
		t.Fatal(err)
	}

	rows = schedule(7, "seven")

	full, err := options.NewRead(ctx, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	out, err := ws.ReadWorkingSchedule(ctx, full)
	if err != nil {
		t.Fatal(err)
	}

	if out.Name != "seven" {
		t.Errorf("full read returned %q: the field-masked read poisoned the cache", out.Name)
	}
}

// The cache key carries the domain, so one tenant can never be served another
// tenant's row.
func TestReadWorkingScheduleIsolatesDomains(t *testing.T) {
	rows := schedule(7, "domain one")

	ws, _ := newProbe(t, &rows, &rows, 7)

	one := userCtx(1, 1)

	read, err := options.NewRead(one, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ws.ReadWorkingSchedule(one, read); err != nil {
		t.Fatal(err)
	}

	rows = schedule(7, "domain two")

	two := userCtx(2, 1)

	read2, err := options.NewRead(two, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	out, err := ws.ReadWorkingSchedule(two, read2)
	if err != nil {
		t.Fatal(err)
	}

	if out.Name != "domain two" {
		t.Errorf("domain 2 read returned %q: it was served domain 1's cache entry", out.Name)
	}
}

func TestDeleteWorkingScheduleInvalidatesItem(t *testing.T) {
	rows := schedule(7, "seven")

	ws, _ := newProbe(t, &rows, &rows, 7)
	ctx := userCtx(1, 1)

	read, err := options.NewRead(ctx, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ws.ReadWorkingSchedule(ctx, read); err != nil {
		t.Fatal(err)
	}

	if _, ok, _ := ws.items.Get(ctx, itemKey(1, 7)); !ok {
		t.Fatal("read did not populate the entity cache")
	}

	if _, err := ws.DeleteWorkingSchedule(ctx, read); err != nil {
		t.Fatal(err)
	}

	if _, ok, _ := ws.items.Get(ctx, itemKey(1, 7)); ok {
		t.Error("delete left the entity in the cache")
	}
}

// Searches are not cached at all: no collection key can encode the filters or
// the paging of the query that produced it.
func TestSearchWorkingScheduleIsNotCached(t *testing.T) {
	rows := schedule(7, "night")

	ws, _ := newProbe(t, &rows, &rows, 7)
	ctx := userCtx(1, 1)

	filtered, err := options.NewSearch(ctx, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ws.SearchWorkingSchedule(ctx, filtered); err != nil {
		t.Fatal(err)
	}

	rows = []*model.WorkingSchedule{
		{DomainRecord: model.DomainRecord{Id: 7}, Name: "night"},
		{DomainRecord: model.DomainRecord{Id: 8}, Name: "day"},
	}

	all, err := options.NewSearch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	out, err := ws.SearchWorkingSchedule(ctx, all)
	if err != nil {
		t.Fatal(err)
	}

	if len(out) != 2 {
		t.Fatalf("unfiltered search returned %d rows, want 2: a narrower search was served from a cache", len(out))
	}
}
