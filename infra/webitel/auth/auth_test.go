package auth

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authclient "buf.build/gen/go/webitel/webitel-go/grpc/go/_gogrpc"
	authmodel "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// fakeAuth answers UserInfo from a callback and counts the calls.
type fakeAuth struct {
	authclient.AuthClient

	calls atomic.Int64
	fn    func(ctx context.Context) (*authmodel.Userinfo, error)
}

func (f *fakeAuth) UserInfo(ctx context.Context, _ *authmodel.UserinfoRequest, _ ...grpc.CallOption) (*authmodel.Userinfo, error) {
	f.calls.Add(1)

	return f.fn(ctx)
}

func newClient(fn func(ctx context.Context) (*authmodel.Userinfo, error)) (*Client, *fakeAuth) {
	api := &fakeAuth{fn: fn}

	return &Client{api: api}, api
}

// The session is resolved on every request. Caching it would keep a revoked
// token working until the entry expired.
func TestSessionIsNotCached(t *testing.T) {
	cli, api := newClient(func(context.Context) (*authmodel.Userinfo, error) {
		return &authmodel.Userinfo{UserId: 3, Dc: 1}, nil
	})

	for range 3 {
		if _, err := cli.Session(context.Background(), "token"); err != nil {
			t.Fatal(err)
		}
	}

	if got := api.calls.Load(); got != 3 {
		t.Errorf("auth service called %d times for 3 requests, want 3: a session was reused", got)
	}
}

// Concurrent requests carrying the same token collapse into one lookup.
func TestSessionCollapsesConcurrentLookups(t *testing.T) {
	release := make(chan struct{})
	cli, api := newClient(func(context.Context) (*authmodel.Userinfo, error) {
		<-release

		return &authmodel.Userinfo{UserId: 3, Dc: 1}, nil
	})

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, err := cli.Session(context.Background(), "token"); err != nil {
				t.Error(err)
			}
		}()
	}

	// Give the goroutines time to queue up behind the same key.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := api.calls.Load(); got != 1 {
		t.Errorf("auth service called %d times for 8 concurrent requests, want 1", got)
	}
}

// A client that hangs up must not fail the requests waiting on the same lookup:
// its context is the one the shared call would otherwise inherit.
func TestSessionLeaderCancellationDoesNotFailWaiters(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	cli, _ := newClient(func(ctx context.Context) (*authmodel.Userinfo, error) {
		close(started)
		<-release

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		return &authmodel.Userinfo{UserId: 3, Dc: 1}, nil
	})

	leader, cancel := context.WithCancel(context.Background())

	go func() {
		_, _ = cli.Session(leader, "token")
	}()

	<-started

	waiter := make(chan error, 1)

	go func() {
		_, err := cli.Session(context.Background(), "token")
		waiter <- err
	}()

	// The leader gives up while the waiter is still queued behind it.
	time.Sleep(50 * time.Millisecond)
	cancel()
	close(release)

	select {
	case err := <-waiter:
		if err != nil {
			t.Fatalf("a healthy request failed because another client hung up: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the waiting request never returned")
	}
}

func TestSessionErrors(t *testing.T) {
	t.Run("unauthenticated is reported as such", func(t *testing.T) {
		cli, _ := newClient(func(context.Context) (*authmodel.Userinfo, error) {
			return nil, status.Error(codes.Unauthenticated, "bad token")
		})

		_, err := cli.Session(context.Background(), "token")
		if !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("err = %v, want ErrUnauthenticated", err)
		}
	})

	t.Run("a service outage is not an authentication failure", func(t *testing.T) {
		cli, _ := newClient(func(context.Context) (*authmodel.Userinfo, error) {
			return nil, status.Error(codes.Unavailable, "connection refused")
		})

		_, err := cli.Session(context.Background(), "token")
		if !errors.Is(err, ErrInternal) {
			t.Errorf("err = %v, want ErrInternal", err)
		}
	})

	t.Run("an empty response is rejected", func(t *testing.T) {
		cli, _ := newClient(func(context.Context) (*authmodel.Userinfo, error) {
			return nil, nil //nolint:nilnil // the malformed answer under test
		})

		if _, err := cli.Session(context.Background(), "token"); err == nil {
			t.Error("an empty user info produced a session")
		}
	})

	t.Run("a failed lookup is retried, never remembered", func(t *testing.T) {
		var fail atomic.Bool

		fail.Store(true)

		cli, _ := newClient(func(context.Context) (*authmodel.Userinfo, error) {
			if fail.Load() {
				return nil, status.Error(codes.Unavailable, "down")
			}

			return &authmodel.Userinfo{UserId: 3, Dc: 1}, nil
		})

		if _, err := cli.Session(context.Background(), "token"); err == nil {
			t.Fatal("expected the first lookup to fail")
		}

		fail.Store(false)

		if _, err := cli.Session(context.Background(), "token"); err != nil {
			t.Errorf("the recovered lookup still failed: %v", err)
		}
	})
}

// Two callers with different tokens must never share a session.
func TestSessionKeyedByToken(t *testing.T) {
	cli, _ := newClient(func(ctx context.Context) (*authmodel.Userinfo, error) {
		// The token travels as outgoing metadata; echo it back as the user id.
		if tokenOf(ctx) == "second" {
			return &authmodel.Userinfo{UserId: 2, Dc: 1}, nil
		}

		return &authmodel.Userinfo{UserId: 1, Dc: 1}, nil
	})

	first, err := cli.Session(context.Background(), "first")
	if err != nil {
		t.Fatal(err)
	}

	second, err := cli.Session(context.Background(), "second")
	if err != nil {
		t.Fatal(err)
	}

	if first.UserID == second.UserID {
		t.Errorf("both tokens resolved to user %d", first.UserID)
	}

	if first.Token != "first" || second.Token != "second" {
		t.Errorf("sessions carry the wrong tokens: %q, %q", first.Token, second.Token)
	}
}

func tokenOf(ctx context.Context) string {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ""
	}

	if v := md.Get(hdrToken); len(v) > 0 {
		return v[0]
	}

	return ""
}
