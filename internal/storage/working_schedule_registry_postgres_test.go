package storage

import (
	"context"
	"slices"
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/internal/model/options"
)

func TestSearchWorkingSchedulesSortsOnPostgres(t *testing.T) {
	db, _ := newPostgres(t)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}

	cases := []struct {
		sort string
		want []int64
	}{
		{sort: "", want: []int64{1, 2, 3, 4}},
		{sort: "+state", want: []int64{2, 1, 3, 4}},
		{sort: "-state", want: []int64{4, 3, 1, 2}},
		{sort: "+name", want: []int64{2, 4, 1, 3}},
		{sort: "-name", want: []int64{3, 1, 4, 2}},
	}

	for _, c := range cases {
		t.Run(c.sort, func(t *testing.T) {
			search, err := options.NewSearch(ctx, options.WithOrder(c.sort))
			if err != nil {
				t.Fatal(err)
			}

			items, err := s.SearchWorkingSchedule(ctx, search)
			if err != nil {
				t.Fatal(err)
			}

			got := make([]int64, 0, len(items))
			for _, item := range items {
				got = append(got, item.ID)
			}

			if !slices.Equal(got, c.want) {
				t.Errorf("order %v, want %v", got, c.want)
			}
		})
	}
}

func TestSearchWorkingSchedulesRefusesOtherSortsOnPostgres(t *testing.T) {
	db, _ := newPostgres(t)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}

	for _, sort := range []string{"+id", "-created_at", "name; SELECT 1", "+(SELECT pg_sleep(1))", "-state DESC, id"} {
		t.Run(sort, func(t *testing.T) {
			search, err := options.NewSearch(ctx, options.WithOrder(sort))
			if err != nil {
				t.Fatal(err)
			}

			if _, err := s.SearchWorkingSchedule(ctx, search); !errors.Is(err, ErrWorkingScheduleSort) {
				t.Errorf("err = %v, want %v", err, ErrWorkingScheduleSort)
			}
		})
	}
}

func TestWorkingScheduleCountsItsAgentsOnPostgres(t *testing.T) {
	db, _ := newPostgres(t)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}
	want := map[int64]int64{1: 3, 2: 1, 3: 2, 4: 1}

	for _, mask := range [][]string{nil, {"id", "name"}, {"id", "agents"}} {
		search, err := options.NewSearch(ctx, options.WithFields(mask))
		if err != nil {
			t.Fatal(err)
		}

		items, err := s.SearchWorkingSchedule(ctx, search)
		if err != nil {
			t.Fatalf("%v: %v", mask, err)
		}

		if len(items) != len(want) {
			t.Fatalf("%v: %d schedules, want %d", mask, len(items), len(want))
		}

		for _, item := range items {
			if got := item.MarshalProto().GetTotalAgents(); got != want[item.ID] {
				t.Errorf("%v: schedule %d has %d agents, want %d", mask, item.ID, got, want[item.ID])
			}
		}
	}

	read, err := options.NewRead(ctx, options.WithID(3), options.WithFields([]string{"name"}))
	if err != nil {
		t.Fatal(err)
	}

	item, err := s.ReadWorkingSchedule(ctx, read)
	if err != nil {
		t.Fatal(err)
	}

	if item.TotalAgents != 2 {
		t.Errorf("read: %d agents, want 2", item.TotalAgents)
	}
}

func TestRemoveAgentTakesTheirShiftsOnPostgres(t *testing.T) {
	db, conn := newPostgres(t)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}

	count := func(sql string) int {
		t.Helper()

		var n int
		if err := conn.QueryRow(context.Background(), sql).Scan(&n); err != nil {
			t.Fatal(err)
		}

		return n
	}

	other, err := options.NewRead(userCtx(2, 2), options.WithID(2))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.UpdateWorkingScheduleRemoveAgent(userCtx(2, 2), other, 1); err != nil {
		t.Fatal(err)
	}

	if n := count("SELECT count(*) FROM wfm.working_schedule_agent WHERE working_schedule_id = 2 AND agent_id = 1"); n != 1 {
		t.Fatalf("another domain removed the agent")
	}

	read, err := options.NewRead(ctx, options.WithID(2))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.UpdateWorkingScheduleRemoveAgent(ctx, read, 1); err != nil {
		t.Fatal(err)
	}

	checks := map[string]int{
		"SELECT count(*) FROM wfm.working_schedule_agent WHERE working_schedule_id = 2 AND agent_id = 1": 0,
		"SELECT count(*) FROM wfm.agent_working_schedule WHERE working_schedule_agent_id = 3":            0,
		"SELECT count(*) FROM wfm.agent_working_schedule WHERE working_schedule_agent_id = 4":            1,
		"SELECT count(*) FROM wfm.working_schedule_agent WHERE agent_id = 1":                             3,
		"SELECT count(*) FROM wfm.agent_absence":                                                         2,
	}

	for sql, want := range checks {
		if n := count(sql); n != want {
			t.Errorf("%s = %d, want %d", sql, n, want)
		}
	}
}
