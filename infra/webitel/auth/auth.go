package auth

import (
	"context"
	"time"

	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authclient "buf.build/gen/go/webitel/webitel-go/grpc/go/_gogrpc"
	authmodel "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/webitel"
)

const (
	serviceName = "go.webitel.app"
	hdrToken    = "x-webitel-access"

	// lookupTimeout caps a single user info request.
	lookupTimeout = 15 * time.Second
)

var (
	ErrUnauthenticated = errors.Unauthenticated("authentication failed", errors.WithID("webitel.auth.unauthenticated"))
	ErrInternal        = errors.Internal("auth service is unavailable", errors.WithID("webitel.auth.internal"))
)

// Manager resolves the session behind an access token.
type Manager interface {
	Session(ctx context.Context, token string) (*Session, error)
}

// Client resolves sessions from the Webitel auth service, collapsing
// concurrent lookups of the same token into one request. Sessions are
// deliberately not cached: a cached session would keep a revoked token working
// until it expired.
type Client struct {
	conn  *grpc.ClientConn
	api   authclient.AuthClient
	group singleflight.Group
}

func New(dp discovery.Discovery) (*Client, error) {
	// Fail fast: a missing auth service must not stall every request.
	conn, err := webitel.Dial(dp, serviceName, grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [ { "round_robin": {} } ]}`))
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn, api: authclient.NewAuthClient(conn)}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) HealthCheck(context.Context) error {
	state := c.conn.GetState()
	if state != connectivity.Idle && state != connectivity.Ready {
		return errors.New("service is not ready", errors.WithValue("state", state.String()))
	}

	return nil
}

func (c *Client) Session(ctx context.Context, token string) (*Session, error) {
	// The lookup is detached from the caller that happens to lead it: one
	// client hanging up must not fail everyone waiting on the same token.
	ch := c.group.DoChan(token, func() (any, error) {
		return c.fetch(context.WithoutCancel(ctx), token)
	})

	select {
	case <-ctx.Done():
		return nil, errors.Wrap(ErrInternal, errors.WithCause(ctx.Err()))
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}

		s, ok := res.Val.(*Session)
		if !ok {
			return nil, ErrInternal
		}

		return s, nil
	}
}

func (c *Client) fetch(ctx context.Context, token string) (*Session, error) {
	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(ctx, metadata.Pairs(hdrToken, token)), lookupTimeout)
	defer cancel()

	info, err := c.api.UserInfo(ctx, &authmodel.UserinfoRequest{})
	if err != nil {
		if status.Code(err) == codes.Unauthenticated {
			return nil, errors.Wrap(ErrUnauthenticated, errors.WithCause(err))
		}

		return nil, errors.Wrap(ErrInternal, errors.WithCause(err))
	}

	if info == nil {
		return nil, ErrUnauthenticated
	}

	return NewSession(token, info), nil
}
