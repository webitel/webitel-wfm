package engine

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	gogrpc "buf.build/gen/go/webitel/engine/grpc/go/_gogrpc"
	pb "buf.build/gen/go/webitel/engine/protocolbuffers/go"

	"github.com/webitel/webitel-wfm/internal/model"
)

type pagedAgents struct {
	gogrpc.AgentServiceClient

	total      int
	emptyAfter int
	calls      int
}

func (f *pagedAgents) SearchAgent(_ context.Context, in *pb.SearchAgentRequest, _ ...grpc.CallOption) (*pb.ListAgent, error) {
	f.calls++

	if f.emptyAfter > 0 && f.calls > f.emptyAfter {
		return &pb.ListAgent{Next: true}, nil
	}

	size, page := int(in.GetSize()), int(max(in.GetPage(), 1))
	if size <= 0 {
		size = 40
	}

	from := (page - 1) * size
	to := min(from+size, f.total)

	out := &pb.ListAgent{Next: to < f.total}
	for id := from; id < to; id++ {
		out.Items = append(out.Items, &pb.Agent{Id: int64(id + 1)})
	}

	return out, nil
}

func TestAgentsReadsEveryPage(t *testing.T) {
	cases := []struct {
		name       string
		total      int
		emptyAfter int
		want       int
		calls      int
	}{
		{name: "none", total: 0, want: 0, calls: 1},
		{name: "more than engine's default of 40", total: 41, want: 41, calls: 1},
		{name: "a page exactly", total: agentsPageSize, want: agentsPageSize, calls: 1},
		{name: "several pages", total: 2*agentsPageSize + 1, want: 2*agentsPageSize + 1, calls: 3},
		{name: "an empty page that says there is more", total: 3 * agentsPageSize, emptyAfter: 1, want: agentsPageSize, calls: 2},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &pagedAgents{total: c.total, emptyAfter: c.emptyAfter}

			ids, err := (&AgentService{cli: fake}).Agents(context.Background(), &model.AgentSearch{TeamIDs: []int64{1}})
			if err != nil {
				t.Fatal(err)
			}

			if len(ids) != c.want || fake.calls != c.calls {
				t.Fatalf("%d agents in %d calls, want %d in %d", len(ids), fake.calls, c.want, c.calls)
			}

			for i, id := range ids {
				if id != int64(i+1) {
					t.Fatalf("agent %d is %d, want %d", i, id, i+1)
				}
			}
		})
	}
}
