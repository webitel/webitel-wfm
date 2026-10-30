package service

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"google.golang.org/grpc"

	gogrpc "buf.build/gen/go/webitel/engine/grpc/go/_gogrpc"
	enginepb "buf.build/gen/go/webitel/engine/protocolbuffers/go"

	"github.com/webitel/webitel-go-kit/infra/discovery"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
)

// fakeEngine answers the engine calls the services make with canned data.
type fakeEngine struct {
	gogrpc.UnimplementedAgentServiceServer
	gogrpc.UnimplementedAgentTeamServiceServer

	agents []int64
	team   *enginepb.AgentTeam

	agentReads atomic.Int32
}

func (f *fakeEngine) ReadAgent(_ context.Context, in *enginepb.ReadAgentRequest) (*enginepb.Agent, error) {
	f.agentReads.Add(1)

	return &enginepb.Agent{Id: in.GetId()}, nil
}

func (f *fakeEngine) SearchAgent(context.Context, *enginepb.SearchAgentRequest) (*enginepb.ListAgent, error) {
	items := make([]*enginepb.Agent, 0, len(f.agents))
	for _, id := range f.agents {
		items = append(items, &enginepb.Agent{Id: id})
	}

	return &enginepb.ListAgent{Items: items}, nil
}

func (f *fakeEngine) ReadAgentTeam(context.Context, *enginepb.ReadAgentTeamRequest) (*enginepb.AgentTeam, error) {
	return f.team, nil
}

// staticDiscovery resolves every service to one fixed endpoint.
type staticDiscovery struct{ endpoint string }

func (d staticDiscovery) instances() []*discovery.ServiceInstance {
	return []*discovery.ServiceInstance{{Id: "1", Name: "engine", Endpoints: []string{d.endpoint}}}
}

func (d staticDiscovery) GetService(context.Context, string) ([]*discovery.ServiceInstance, error) {
	return d.instances(), nil
}

func (d staticDiscovery) GetWatcher(context.Context, string) (discovery.Watcher, error) {
	return &staticWatcher{instances: d.instances(), stop: make(chan struct{})}, nil
}

// staticWatcher reports its instances once, then blocks until stopped.
type staticWatcher struct {
	instances []*discovery.ServiceInstance
	sent      bool
	stop      chan struct{}
	once      sync.Once
}

func (w *staticWatcher) Next() ([]*discovery.ServiceInstance, error) {
	if !w.sent {
		w.sent = true

		return w.instances, nil
	}

	<-w.stop

	return nil, context.Canceled
}

func (w *staticWatcher) Stop() error {
	w.once.Do(func() { close(w.stop) })

	return nil
}

// newEngine serves fake over a local listener and dials it the way the service
// dials the real engine.
func newEngine(t *testing.T, fake *fakeEngine) *engine.Client {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv := grpc.NewServer()
	gogrpc.RegisterAgentServiceServer(srv, fake)
	gogrpc.RegisterAgentTeamServiceServer(srv, fake)

	go func() { _ = srv.Serve(lis) }()

	t.Cleanup(srv.Stop)

	cli, err := engine.New(staticDiscovery{endpoint: "grpc://" + lis.Addr().String()})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = cli.Close() })

	return cli
}
