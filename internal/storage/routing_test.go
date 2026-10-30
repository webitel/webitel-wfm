package storage

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
)

// trace is what the probes record: the role that answered each query, in
// order, plus the statements themselves, so a test can assert what was asked
// and not only where.
type trace struct {
	asked  []string
	calls  []call
	queued int
}

type call struct {
	sql  string
	args []any
}

// probeNode answers Select and Get with canned rows and records that it was
// asked, so a test can tell which role of the cluster a query was routed to.
type probeNode[T any] struct {
	dbsql.Node

	role string
	rows *[]*T
	id   int64
	ids  []int64 // what a batch or a multi-row INSERT on this node returns
	tr   *trace
}

func (p probeNode[T]) Select(_ context.Context, dest any, sql string, args ...any) error {
	p.tr.asked = append(p.tr.asked, p.role)
	p.tr.calls = append(p.tr.calls, call{sql: sql, args: args})

	switch out := dest.(type) {
	case *[]*T:
		if p.rows != nil {
			*out = *p.rows
		}
	case *[]int64: // a multi-row INSERT ... RETURNING id
		*out = append(*out, p.ids...)
	}

	return nil
}

func (p probeNode[T]) Get(_ context.Context, dest any, sql string, args ...any) error {
	p.tr.asked = append(p.tr.asked, p.role)
	p.tr.calls = append(p.tr.calls, call{sql: sql, args: args})

	switch out := dest.(type) {
	case *int64:
		*out = p.id
	case *T:
		if p.rows != nil && len(*p.rows) > 0 {
			*out = *(*p.rows)[0]
		}
	case **string: // a name lookup such as to_regproc: found
		name := "found"
		*out = &name
	}

	return nil
}

func (p probeNode[T]) Exec(_ context.Context, sql string, args ...any) error {
	p.tr.asked = append(p.tr.asked, p.role)
	p.tr.calls = append(p.tr.calls, call{sql: sql, args: args})

	return nil
}

func (p probeNode[T]) Batch() dbsql.BatchNode {
	return probeBatch{role: p.role, ids: p.ids, tr: p.tr}
}

// probeBatch counts what was queued and answers Select with canned ids, the
// way the real batch appends one result set per queued query.
type probeBatch struct {
	role string
	ids  []int64
	tr   *trace
}

func (b probeBatch) Queue(string, ...any) { b.tr.queued++ }

func (b probeBatch) Select(_ context.Context, dest any) error {
	b.tr.asked = append(b.tr.asked, b.role)

	if out, ok := dest.(*[]int64); ok {
		*out = append(*out, b.ids...)
	}

	return nil
}

// probeStore embeds the interface so the fake keeps compiling as the interface
// grows; everything the code under test calls is overridden.
type probeStore[T any] struct {
	dbsql.Store

	primary probeNode[T]
	standby probeNode[T]
}

func (p *probeStore[T]) Primary() dbsql.Node          { return p.primary }
func (p *probeStore[T]) StandbyPreferred() dbsql.Node { return p.standby }

// newProbeStore wires nodes onto canned rows. primaryRows and standbyRows are
// separate so a test can simulate replication lag; batchIDs is what a batch or
// a multi-row INSERT on either node returns.
func newProbeStore[T any](primaryRows, standbyRows *[]*T, id int64, batchIDs ...int64) (*probeStore[T], *trace) {
	tr := &trace{}
	db := &probeStore[T]{
		primary: probeNode[T]{role: "primary", rows: primaryRows, id: id, ids: batchIDs, tr: tr},
		standby: probeNode[T]{role: "standby", rows: standbyRows, id: id, ids: batchIDs, tr: tr},
	}

	return db, tr
}

func newProbe(t *testing.T, primaryRows, standbyRows *[]*model.WorkingSchedule, id int64) (*WorkingSchedule, *trace) {
	t.Helper()

	db, tr := newProbeStore(primaryRows, standbyRows, id)

	return &WorkingSchedule{db: db}, tr
}

func newCalendarProbe(t *testing.T, batchIDs ...int64) (*AgentWorkingSchedule, *trace) {
	t.Helper()

	db, tr := newProbeStore[model.AgentWorkingSchedule](nil, nil, 0, batchIDs...)

	return &AgentWorkingSchedule{db: db}, tr
}

func assertAsked(t *testing.T, asked []string, want ...string) {
	t.Helper()

	if !slices.Equal(asked, want) {
		t.Errorf("queries went to %v, want %v", asked, want)
	}
}

func userCtx(domain, user int64) context.Context {
	return grpccontext.SetUser(context.Background(), &model.SignedInUser{Id: user, DomainId: domain})
}

func schedule(id int64, name string) []*model.WorkingSchedule {
	return []*model.WorkingSchedule{{DomainRecord: model.DomainRecord{Id: id}, Name: name}}
}

// A row read straight after a write must come from the primary: a standby may
// not have replicated it yet.
func TestUpdateWorkingScheduleReadsBackFromPrimary(t *testing.T) {
	primary := schedule(7, "renamed")
	standby := schedule(7, "stale")

	ws, tr := newProbe(t, &primary, &standby, 7)
	ctx := userCtx(1, 1)

	in := &model.WorkingSchedule{DomainRecord: model.DomainRecord{Id: 7}, Name: "renamed"}

	out, err := ws.UpdateWorkingSchedule(ctx, &model.SignedInUser{Id: 1, DomainId: 1}, in)
	if err != nil {
		t.Fatal(err)
	}

	if out.Name != "renamed" {
		t.Errorf("update returned %q, want %q: the read-back went to a lagging standby", out.Name, "renamed")
	}

	// The UPDATE, then the read-back.
	assertAsked(t, tr.asked, "primary", "primary")
}

func TestCreateWorkingScheduleReadsBackFromPrimary(t *testing.T) {
	primary := schedule(9, "created")

	var standby []*model.WorkingSchedule // not replicated yet

	ws, tr := newProbe(t, &primary, &standby, 9)
	ctx := userCtx(1, 1)

	in := &model.WorkingSchedule{Name: "created"}

	out, err := ws.CreateWorkingSchedule(ctx, &model.SignedInUser{Id: 1, DomainId: 1}, in)
	if err != nil {
		t.Fatalf("create of a committed row failed: %v", err)
	}

	if out.Name != "created" {
		t.Errorf("create returned %q, want %q", out.Name, "created")
	}

	// The INSERT, then the read-back.
	assertAsked(t, tr.asked, "primary", "primary")
}

func TestUpdateWorkingScheduleAddAgentsReadsBackFromPrimary(t *testing.T) {
	primary := []*model.WorkingSchedule{{DomainRecord: model.DomainRecord{Id: 7}, Agents: []*model.LookupItem{{Id: 3}}}}
	standby := schedule(7, "stale") // the agent row is not replicated yet

	ws, tr := newProbe(t, &primary, &standby, 7)
	ctx := userCtx(1, 1)

	read, err := options.NewRead(ctx, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	out, err := ws.UpdateWorkingScheduleAddAgents(ctx, read, []int64{3})
	if err != nil {
		t.Fatal(err)
	}

	if len(out) != 1 || out[0].Id != 3 {
		t.Errorf("add agents returned %+v, want the agent just added: the read-back went to a lagging standby", out)
	}

	// The INSERT, then the read-back.
	assertAsked(t, tr.asked, "primary", "primary")
}

// A client rereads the schedule by id straight after saving, so that read
// must not reach a lagging standby either.
func TestReadWorkingScheduleReadsFromPrimary(t *testing.T) {
	primary := schedule(7, "renamed")
	standby := schedule(7, "stale")

	ws, tr := newProbe(t, &primary, &standby, 7)
	ctx := userCtx(1, 1)

	read, err := options.NewRead(ctx, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	out, err := ws.ReadWorkingSchedule(ctx, read)
	if err != nil {
		t.Fatal(err)
	}

	if out.Name != "renamed" {
		t.Errorf("read returned %q, want %q: it went to a lagging standby", out.Name, "renamed")
	}

	assertAsked(t, tr.asked, "primary")
}

// With no cache in front of it, the query alone keeps one tenant from reading
// another tenant's row.
func TestReadWorkingScheduleScopesByDomainAndID(t *testing.T) {
	rows := schedule(7, "seven")

	ws, tr := newProbe(t, &rows, &rows, 7)
	ctx := userCtx(42, 1)

	read, err := options.NewRead(ctx, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ws.ReadWorkingSchedule(ctx, read); err != nil {
		t.Fatal(err)
	}

	if len(tr.calls) != 1 {
		t.Fatalf("%d queries, want 1", len(tr.calls))
	}

	c := tr.calls[0]
	if !strings.Contains(c.sql, "domain_id") || !slices.Contains(c.args, any(int64(42))) {
		t.Errorf("read is not scoped to the caller's domain: %s %v", c.sql, c.args)
	}

	if !slices.Contains(c.args, any(int64(7))) {
		t.Errorf("read is not filtered by id: %s %v", c.sql, c.args)
	}
}

func TestReadWorkingScheduleFieldMaskSelectsOnlyThoseColumns(t *testing.T) {
	rows := []*model.WorkingSchedule{{DomainRecord: model.DomainRecord{Id: 7}}}

	ws, tr := newProbe(t, &rows, &rows, 7)
	ctx := userCtx(1, 1)

	masked, err := options.NewRead(ctx, options.WithID(7), options.WithFields([]string{"id"}))
	if err != nil {
		t.Fatal(err)
	}

	out, err := ws.ReadWorkingSchedule(ctx, masked)
	if err != nil {
		t.Fatal(err)
	}

	if out.Id != 7 {
		t.Errorf("masked read returned id %d, want 7", out.Id)
	}

	if sql := tr.calls[0].sql; !strings.HasPrefix(sql, "SELECT id FROM") {
		t.Errorf("field mask did not reach the SELECT list: %s", sql)
	}
}

// read maps the row count onto the error contract: more than one row under an
// id is a conflict, none is ErrNoRows, and neither comes with a value.
func TestReadWorkingScheduleErrors(t *testing.T) {
	cases := map[string]struct {
		rows []*model.WorkingSchedule
		want error
	}{
		"duplicate rows": {
			rows: []*model.WorkingSchedule{
				{DomainRecord: model.DomainRecord{Id: 7}, Name: "one"},
				{DomainRecord: model.DomainRecord{Id: 7}, Name: "two"},
			},
			want: dbsql.ErrEntityConflict,
		},
		"missing row": {rows: []*model.WorkingSchedule{}, want: dbsql.ErrNoRows},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ws, _ := newProbe(t, &tc.rows, &tc.rows, 7)
			ctx := userCtx(1, 1)

			read, err := options.NewRead(ctx, options.WithID(7))
			if err != nil {
				t.Fatal(err)
			}

			out, err := ws.ReadWorkingSchedule(ctx, read)
			if !errors.Is(err, tc.want) {
				t.Fatalf("read returned (%v, %v), want %v", out, err, tc.want)
			}

			if out != nil {
				t.Errorf("read returned %+v alongside the error", out)
			}
		})
	}
}

// Lists stay on the standby.
func TestSearchWorkingScheduleReadsFromStandby(t *testing.T) {
	rows := schedule(7, "seven")

	ws, tr := newProbe(t, &rows, &rows, 7)
	ctx := userCtx(1, 1)

	search, err := options.NewSearch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ws.SearchWorkingSchedule(ctx, search); err != nil {
		t.Fatal(err)
	}

	assertAsked(t, tr.asked, "standby")
}

func TestWorkingScheduleWritesRunOnPrimary(t *testing.T) {
	ctx := userCtx(1, 1)

	read, err := options.NewRead(ctx, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	writes := map[string]struct {
		call func(*WorkingSchedule) (int64, error)
		want int64
	}{
		"delete": {
			call: func(ws *WorkingSchedule) (int64, error) { return ws.DeleteWorkingSchedule(ctx, read) },
			want: 7,
		},
		"remove agent": {
			call: func(ws *WorkingSchedule) (int64, error) { return ws.UpdateWorkingScheduleRemoveAgent(ctx, read, 3) },
			want: 3,
		},
	}

	for name, tc := range writes {
		t.Run(name, func(t *testing.T) {
			rows := schedule(7, "seven")
			ws, tr := newProbe(t, &rows, &rows, 7)

			got, err := tc.call(ws)
			if err != nil {
				t.Fatal(err)
			}

			if got != tc.want {
				t.Errorf("returned %d, want %d", got, tc.want)
			}

			assertAsked(t, tr.asked, "primary")
		})
	}
}

// The calendar content, shifts and the holidays returned with them, is reread
// after every edit.
func TestCalendarReadsFromPrimary(t *testing.T) {
	user := &model.SignedInUser{Id: 1, DomainId: 1}
	search := &model.AgentWorkingScheduleSearch{
		WorkingScheduleId: 7,
		SearchItem:        model.SearchItem{Date: &model.FilterBetween{}},
	}

	reads := map[string]func(*AgentWorkingSchedule) error{
		"shifts": func(a *AgentWorkingSchedule) error {
			_, err := a.SearchAgentWorkingSchedule(context.Background(), user, search)

			return err
		},
		"holidays": func(a *AgentWorkingSchedule) error {
			_, err := a.Holidays(context.Background(), user, search)

			return err
		},
	}

	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			aws, tr := newCalendarProbe(t)
			if err := read(aws); err != nil {
				t.Fatal(err)
			}

			assertAsked(t, tr.asked, "primary")
		})
	}
}

// Shift create writes one INSERT per shift and rereads exactly the rows it
// wrote, from the primary.
func TestCreateAgentsWorkingScheduleShiftsRereadsTheRowsItWrote(t *testing.T) {
	aws, tr := newCalendarProbe(t, 11, 12)

	in := []*model.AgentWorkingSchedule{{
		Agent: model.LookupItem{Id: 3},
		Schedule: []*model.AgentSchedule{
			{Shift: &model.AgentScheduleShift{Start: 540, End: 1080}},
			{Shift: &model.AgentScheduleShift{Start: 600, End: 900}},
		},
	}}

	if _, err := aws.CreateAgentsWorkingScheduleShifts(context.Background(), &model.SignedInUser{Id: 1, DomainId: 1}, 7, in); err != nil {
		t.Fatal(err)
	}

	if tr.queued != 2 {
		t.Errorf("queued %d inserts, want one per shift (2)", tr.queued)
	}

	// The batch, then the read-back of the rows it wrote.
	assertAsked(t, tr.asked, "primary", "primary")

	if len(tr.calls) != 1 {
		t.Fatalf("%d read-back queries, want 1", len(tr.calls))
	}

	c := tr.calls[0]
	if !strings.Contains(c.sql, "(shift ->> 'id')::bigint") {
		t.Errorf("read-back is not filtered to the rows written: %s", c.sql)
	}

	for _, id := range []int64{11, 12} {
		if !slices.Contains(c.args, any(id)) {
			t.Errorf("read-back args %v lack id %d returned by the batch", c.args, id)
		}
	}
}

// assertReadsBackFromPrimary runs a lookup write against a primary holding the
// saved row and a standby still holding a stale one. The write must answer with
// the saved row, read by the id it wrote.
func assertReadsBackFromPrimary[T any](t *testing.T, saved, stale *T, write func(db dbsql.Store) (*T, error)) {
	t.Helper()

	primary, standby := []*T{saved}, []*T{stale}
	db, tr := newProbeStore(&primary, &standby, 7)

	out, err := write(db)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(out, saved) {
		t.Errorf("write returned %+v, want %+v: the read-back went to a lagging standby", out, saved)
	}

	// The write, then the read-back.
	assertAsked(t, tr.asked, "primary", "primary")

	if len(tr.calls) != 2 {
		t.Fatalf("%d queries, want 2", len(tr.calls))
	}

	if back := tr.calls[1]; !slices.Contains(back.args, any(int64(7))) {
		t.Errorf("read-back is not filtered by the id written: %s %v", back.sql, back.args)
	}
}

// A lookup's create and update answer with the row they wrote, read back from
// the primary: a standby may not have it yet, or may still hold the old one.
func TestLookupWritesReadBackFromPrimary(t *testing.T) {
	ctx := userCtx(1, 1)
	user := &model.SignedInUser{Id: 1, DomainId: 1}
	updated := model.DomainRecord{Id: 7}

	t.Run("pause template", func(t *testing.T) {
		saved, stale := &model.PauseTemplate{Name: "saved"}, &model.PauseTemplate{Name: "stale"}
		causes := []model.PauseTemplateCause{{Cause: &model.LookupItem{Id: 3}}}

		assertReadsBackFromPrimary(t, saved, stale, func(db dbsql.Store) (*model.PauseTemplate, error) {
			return (&PauseTemplate{db: db}).CreatePauseTemplate(ctx, user, &model.PauseTemplate{Causes: causes})
		})

		assertReadsBackFromPrimary(t, saved, stale, func(db dbsql.Store) (*model.PauseTemplate, error) {
			return (&PauseTemplate{db: db}).UpdatePauseTemplate(ctx, user, &model.PauseTemplate{DomainRecord: updated, Causes: causes})
		})
	})

	t.Run("shift template", func(t *testing.T) {
		saved, stale := &model.ShiftTemplate{Name: "saved"}, &model.ShiftTemplate{Name: "stale"}

		assertReadsBackFromPrimary(t, saved, stale, func(db dbsql.Store) (*model.ShiftTemplate, error) {
			return (&ShiftTemplate{db: db}).CreateShiftTemplate(ctx, user, &model.ShiftTemplate{})
		})

		assertReadsBackFromPrimary(t, saved, stale, func(db dbsql.Store) (*model.ShiftTemplate, error) {
			return (&ShiftTemplate{db: db}).UpdateShiftTemplate(ctx, user, &model.ShiftTemplate{DomainRecord: updated})
		})
	})

	t.Run("working condition", func(t *testing.T) {
		saved, stale := &model.WorkingCondition{Name: "saved"}, &model.WorkingCondition{Name: "stale"}

		assertReadsBackFromPrimary(t, saved, stale, func(db dbsql.Store) (*model.WorkingCondition, error) {
			return (&WorkingCondition{db: db}).CreateWorkingCondition(ctx, user, &model.WorkingCondition{})
		})

		assertReadsBackFromPrimary(t, saved, stale, func(db dbsql.Store) (*model.WorkingCondition, error) {
			return (&WorkingCondition{db: db}).UpdateWorkingCondition(ctx, user, &model.WorkingCondition{DomainRecord: updated})
		})
	})

	t.Run("agent working conditions", func(t *testing.T) {
		saved := &model.AgentWorkingConditions{WorkingCondition: model.LookupItem{Id: 5}}
		stale := &model.AgentWorkingConditions{WorkingCondition: model.LookupItem{Id: 4}}

		read, err := options.NewRead(ctx, options.WithID(7))
		if err != nil {
			t.Fatal(err)
		}

		assertReadsBackFromPrimary(t, saved, stale, func(db dbsql.Store) (*model.AgentWorkingConditions, error) {
			return (&AgentWorkingConditions{db: db}).UpdateAgentWorkingConditions(ctx, read, saved)
		})
	})
}

// Bulk absence create writes one row per absence and rereads exactly the rows
// it wrote, from the primary.
func TestCreateAgentsAbsencesRereadsTheRowsItWrote(t *testing.T) {
	ctx := userCtx(1, 1)

	primary := []*model.AgentAbsences{{Agent: model.LookupItem{Id: 3}}}

	var standby []*model.AgentAbsences // not replicated yet

	db, tr := newProbeStore(&primary, &standby, 0, 11, 12)

	search, err := options.NewSearch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	in := []*model.AgentAbsences{{Agent: model.LookupItem{Id: 3}, Absence: []*model.Absence{{}, {}}}}

	out, err := (&AgentAbsence{db: db}).CreateAgentsAbsences(ctx, search, in)
	if err != nil {
		t.Fatal(err)
	}

	if len(out) != 1 || out[0].Agent.Id != 3 {
		t.Errorf("create returned %+v, want the absences just written: the read-back went to a lagging standby", out)
	}

	// The INSERT, then the read-back.
	assertAsked(t, tr.asked, "primary", "primary")

	if len(tr.calls) != 2 {
		t.Fatalf("%d queries, want 2", len(tr.calls))
	}

	back := tr.calls[1]
	for _, id := range []int64{11, 12} {
		if !slices.Contains(back.args, any(id)) {
			t.Errorf("read-back args %v lack id %d returned by the insert", back.args, id)
		}
	}
}

func assertStaysOnStandby[T any](t *testing.T, read func(db dbsql.Store) error) {
	t.Helper()

	rows := []*T{new(T)}
	db, tr := newProbeStore(&rows, &rows, 7)

	if err := read(db); err != nil {
		t.Fatal(err)
	}

	assertAsked(t, tr.asked, "standby")
}

// A lookup's plain reads and lists stay on the standby.
func TestLookupReadsStayOnStandby(t *testing.T) {
	ctx := userCtx(1, 1)

	newRead := func(t *testing.T) *options.Read {
		t.Helper()

		read, err := options.NewRead(ctx, options.WithID(7))
		if err != nil {
			t.Fatal(err)
		}

		return read
	}

	newSearch := func(t *testing.T) *options.Search {
		t.Helper()

		search, err := options.NewSearch(ctx, options.WithID(7))
		if err != nil {
			t.Fatal(err)
		}

		return search
	}

	t.Run("pause template", func(t *testing.T) {
		assertStaysOnStandby[model.PauseTemplate](t, func(db dbsql.Store) error {
			_, err := (&PauseTemplate{db: db}).ReadPauseTemplate(ctx, newRead(t))

			return err
		})

		assertStaysOnStandby[model.PauseTemplate](t, func(db dbsql.Store) error {
			_, err := (&PauseTemplate{db: db}).SearchPauseTemplate(ctx, newSearch(t))

			return err
		})
	})

	t.Run("shift template", func(t *testing.T) {
		assertStaysOnStandby[model.ShiftTemplate](t, func(db dbsql.Store) error {
			_, err := (&ShiftTemplate{db: db}).ReadShiftTemplate(ctx, newRead(t))

			return err
		})

		assertStaysOnStandby[model.ShiftTemplate](t, func(db dbsql.Store) error {
			_, err := (&ShiftTemplate{db: db}).SearchShiftTemplate(ctx, newSearch(t))

			return err
		})
	})

	t.Run("working condition", func(t *testing.T) {
		assertStaysOnStandby[model.WorkingCondition](t, func(db dbsql.Store) error {
			_, err := (&WorkingCondition{db: db}).ReadWorkingCondition(ctx, newRead(t))

			return err
		})

		assertStaysOnStandby[model.WorkingCondition](t, func(db dbsql.Store) error {
			_, err := (&WorkingCondition{db: db}).SearchWorkingCondition(ctx, newSearch(t))

			return err
		})
	})

	t.Run("agent working conditions", func(t *testing.T) {
		assertStaysOnStandby[model.AgentWorkingConditions](t, func(db dbsql.Store) error {
			_, err := (&AgentWorkingConditions{db: db}).ReadAgentWorkingConditions(ctx, newRead(t))

			return err
		})
	})

	t.Run("agents absences", func(t *testing.T) {
		assertStaysOnStandby[model.AgentAbsences](t, func(db dbsql.Store) error {
			_, err := (&AgentAbsence{db: db}).SearchAgentsAbsences(ctx, newSearch(t))

			return err
		})
	})
}

// The forecast-calculation store reads its own writes back from the primary
// and the Forecast block loads the procedure it runs from the primary; the
// lookup's own read and list RPCs stay on the standby.
func TestForecastCalculationRouting(t *testing.T) {
	ctx := context.Background()
	user := &model.SignedInUser{Id: 1, DomainId: 1}

	cases := map[string]struct {
		run func(f *ForecastCalculation, tr *trace) (string, error)
		// asked is the sequence of roles; witness must appear in what run
		// returns, and only the right node can produce it.
		asked   []string
		witness string
	}{
		"create reads back": {
			run: func(f *ForecastCalculation, _ *trace) (string, error) {
				out, err := f.CreateForecastCalculation(ctx, user, &model.ForecastCalculation{Procedure: "proc"})
				if err != nil {
					return "", err
				}

				return out.Procedure, nil
			},
			asked:   []string{"primary", "primary", "primary"}, // to_regproc, the INSERT, the read-back
			witness: "proc",
		},
		"update reads back": {
			run: func(f *ForecastCalculation, _ *trace) (string, error) {
				out, err := f.UpdateForecastCalculation(ctx, user, &model.ForecastCalculation{DomainRecord: model.DomainRecord{Id: 7}, Procedure: "proc"})
				if err != nil {
					return "", err
				}

				return out.Procedure, nil
			},
			asked:   []string{"primary", "primary", "primary"}, // to_regproc, the UPDATE, the read-back
			witness: "proc",
		},
		"execute runs the procedure last saved": {
			run: func(f *ForecastCalculation, tr *trace) (string, error) {
				if _, err := f.ExecuteForecastCalculation(ctx, user, 7, 3, &model.FilterBetween{}); err != nil {
					return "", err
				}

				return tr.calls[len(tr.calls)-1].sql, nil
			},
			asked:   []string{"primary", "primary", "primary"}, // the lookup, to_regproc, the procedure
			witness: "FROM proc(",
		},
		"read": {
			run: func(f *ForecastCalculation, _ *trace) (string, error) {
				out, err := f.ReadForecastCalculation(ctx, user, &model.SearchItem{Id: 7})
				if err != nil {
					return "", err
				}

				return out.Procedure, nil
			},
			asked:   []string{"standby"},
			witness: "stale",
		},
		"search": {
			run: func(f *ForecastCalculation, _ *trace) (string, error) {
				_, err := f.SearchForecastCalculation(ctx, user, &model.SearchItem{})

				return "", err
			},
			asked: []string{"standby"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			primary := []*model.ForecastCalculation{{DomainRecord: model.DomainRecord{Id: 7}, Procedure: "proc"}}
			standby := []*model.ForecastCalculation{{DomainRecord: model.DomainRecord{Id: 7}, Procedure: "stale"}}

			db, tr := newProbeStore(&primary, &standby, 7)
			f := &ForecastCalculation{db: db, forecastDB: db}

			got, err := tc.run(f, tr)
			if err != nil {
				t.Fatal(err)
			}

			assertAsked(t, tr.asked, tc.asked...)

			if !strings.Contains(got, tc.witness) {
				t.Errorf("got %q, want it to carry %q from the right node", got, tc.witness)
			}
		})
	}
}
