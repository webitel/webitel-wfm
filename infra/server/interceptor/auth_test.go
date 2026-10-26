package interceptor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	authmodel "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/infra/webitel/auth"
	"github.com/webitel/webitel-wfm/pkg"
)

type sessionFunc func(ctx context.Context, token string) (*auth.Session, error)

func (f sessionFunc) Session(ctx context.Context, token string) (*auth.Session, error) {
	return f(ctx, token)
}

func wfmLicense() []*authmodel.LicenseUser {
	return []*authmodel.LicenseUser{{Prod: "WFM", Scope: []string{"wfm"}}}
}

func TestAuthUnaryServerInterceptor(t *testing.T) {
	handler := func(context.Context, any) (any, error) {
		return "good", nil
	}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/FakeService/FakeMethod",
	}

	type expectation struct {
		err   error
		cause error
		code  codes.Code
	}

	tests := map[string]struct {
		token    *string
		userinfo *authmodel.Userinfo
		expected expectation
	}{
		"empty metadata": {
			token:    nil,
			userinfo: &authmodel.Userinfo{},
			expected: expectation{
				err:  ErrInvalidToken,
				code: codes.Unauthenticated,
			},
		},
		"empty authorization token": {
			token:    pkg.ToPTR(""),
			userinfo: &authmodel.Userinfo{},
			expected: expectation{
				err:   ErrInvalidToken,
				cause: errors.New("empty authorization token"),
				code:  codes.Unauthenticated,
			},
		},
		"empty session": {
			token:    pkg.ToPTR("super-auth-header"),
			userinfo: nil,
			expected: expectation{
				err:   ErrInvalidSession,
				cause: errors.New("empty session"),
				code:  codes.Unauthenticated,
			},
		},
		"session is invalid": {
			token: pkg.ToPTR("super-auth-header"),
			userinfo: &authmodel.Userinfo{
				Dc:        0,
				UserId:    0,
				ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
			},
			expected: expectation{
				err:   ErrInvalidSession,
				cause: errors.New("session has no user"),
				code:  codes.Unauthenticated,
			},
		},
		"authorization token is expired": {
			token: pkg.ToPTR("super-auth-header"),
			userinfo: &authmodel.Userinfo{
				Dc:        1,
				UserId:    3,
				ExpiresAt: time.Now().Truncate(24 * time.Hour).Unix(),
			},
			expected: expectation{
				err:   ErrInvalidSession,
				cause: errors.New("expired authorization token"),
				code:  codes.Unauthenticated,
			},
		},
		"license required": {
			token: pkg.ToPTR("super-auth-header"),
			userinfo: &authmodel.Userinfo{
				Dc:        1,
				UserId:    3,
				ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			},
			expected: expectation{
				err:  ErrLicenseRequired,
				code: codes.PermissionDenied,
			},
		},
	}

	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			am := sessionFunc(func(_ context.Context, token string) (*auth.Session, error) {
				if tt.userinfo == nil {
					return nil, errors.New("empty session")
				}

				return auth.NewSession(token, tt.userinfo), nil
			})

			if tt.token != nil {
				md := metadata.New(map[string]string{
					hdrTokenAccess: *tt.token,
				})

				ctx = metadata.NewIncomingContext(ctx, md)
			}

			_, err := AuthUnaryServerInterceptor(am)(ctx, nil, info, handler)

			// Without this the whole table passes against an interceptor that
			// authenticates nobody.
			require.Error(t, err, "the request was allowed through")

			if tt.expected.cause != nil {
				assert.ErrorContains(t, errors.Cause(err), tt.expected.cause.Error())
			}

			assert.Equal(t, tt.expected.code, errors.Code(err))
			assert.ErrorIs(t, err, tt.expected.err)
		})
	}
}

// The happy path: a valid, licensed session reaches the handler and lands in
// the context the storage layer reads its domain from.
func TestAuthUnaryServerInterceptorAllowsValidSession(t *testing.T) {
	var got *grpccontext.GRPCServerContext

	handler := func(ctx context.Context, _ any) (any, error) {
		got = grpccontext.FromContext(ctx)

		return "good", nil
	}

	am := sessionFunc(func(_ context.Context, token string) (*auth.Session, error) {
		return auth.NewSession(token, &authmodel.Userinfo{
			Dc:        5,
			UserId:    3,
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			License:   wfmLicense(),
		}), nil
	})

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.New(map[string]string{hdrTokenAccess: "super-auth-header"}),
	)

	out, err := AuthUnaryServerInterceptor(am)(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/wfm.PauseTemplateService/ReadPauseTemplate",
	}, handler)

	require.NoError(t, err)
	assert.Equal(t, "good", out)
	require.NotNil(t, got)
	require.NotNil(t, got.SignedInUser, "the handler received no signed in user")

	assert.Equal(t, int64(5), got.SignedInUser.DomainId)
	assert.Equal(t, int64(3), got.SignedInUser.Id)
	assert.Equal(t, "super-auth-header", got.SignedInUser.Token)

	// The object class comes from the generated registry, so a real method name
	// exercises the lookup that "/FakeService/FakeMethod" cannot.
	assert.Equal(t, "wfm_lookups", got.SignedInUser.Object)
}

// Pins what the disabled permission check actually costs: a caller holding no
// grant at all for the object class is still let through. Re-enabling the check
// in auth.go must make this test fail.
func TestAuthUnaryServerInterceptorIgnoresTheObjectClassGrant(t *testing.T) {
	var got *grpccontext.GRPCServerContext

	handler := func(ctx context.Context, _ any) (any, error) {
		got = grpccontext.FromContext(ctx)

		return nil, nil //nolint:nilnil // the handler's answer is irrelevant here
	}

	am := sessionFunc(func(_ context.Context, token string) (*auth.Session, error) {
		return auth.NewSession(token, &authmodel.Userinfo{
			Dc:        5,
			UserId:    3,
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			License:   wfmLicense(),
		}), nil
	})

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.New(map[string]string{hdrTokenAccess: "super-auth-header"}),
	)

	_, err := AuthUnaryServerInterceptor(am)(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/wfm.PauseTemplateService/ReadPauseTemplate",
	}, handler)

	require.NoError(t, err, "the permission check is disabled, so an ungranted caller still reaches the handler")
	require.NotNil(t, got)

	// The check computed a denial and it was discarded, so no row-level rules
	// are requested either.
	assert.False(t, got.SignedInUser.UseRBAC)
}

// A manager that answers with neither a session nor an error must be rejected,
// not dereferenced.
func TestAuthUnaryServerInterceptorNilSession(t *testing.T) {
	am := sessionFunc(func(context.Context, string) (*auth.Session, error) {
		return nil, nil //nolint:nilnil // the malformed answer under test
	})

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.New(map[string]string{hdrTokenAccess: "super-auth-header"}),
	)

	handler := func(context.Context, any) (any, error) { return "good", nil }

	_, err := AuthUnaryServerInterceptor(am)(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/FakeService/FakeMethod",
	}, handler)

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, errors.Code(err))
}
