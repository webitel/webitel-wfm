package interceptor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/webitel/engine/auth_manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	authmock "github.com/webitel/webitel-wfm/gen/go/mocks/infra/webitel/auth"
	"github.com/webitel/webitel-wfm/pkg"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

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
		session  *auth_manager.Session
		expected expectation
	}{
		"empty metadata": {
			token:   nil,
			session: &auth_manager.Session{},
			expected: expectation{
				err:  ErrInvalidToken,
				code: codes.Unauthenticated,
			},
		},
		"empty authorization token": {
			token:   pkg.ToPTR(""),
			session: &auth_manager.Session{},
			expected: expectation{
				err:   ErrInvalidToken,
				cause: werror.New("empty authorization token"),
				code:  codes.Unauthenticated,
			},
		},
		"empty session": {
			token:   pkg.ToPTR("super-auth-header"),
			session: nil,
			expected: expectation{
				err:   ErrInvalidSession,
				cause: werror.New("empty session"),
				code:  codes.Unauthenticated,
			},
		},
		"session is invalid": {
			token: pkg.ToPTR("super-auth-header"),
			session: &auth_manager.Session{
				DomainId: 0,
				UserId:   0,
				Expire:   time.Now().Add(1 * time.Hour).Unix(),
				RoleIds:  []int{},
			},
			expected: expectation{
				err:   ErrInvalidSession,
				cause: werror.New("model.session.is_valid.user_id.app_error"),
				code:  codes.Unauthenticated,
			},
		},
		"authorization token is expired": {
			token: pkg.ToPTR("super-auth-header"),
			session: &auth_manager.Session{
				DomainId: 1,
				UserId:   3,
				Expire:   time.Now().Truncate(24 * time.Hour).Unix(),
				RoleIds:  []int{1},
			},
			expected: expectation{
				err:   ErrInvalidSession,
				cause: werror.New("expired authorization token"),
				code:  codes.Unauthenticated,
			},
		},
		"license required": {
			token: pkg.ToPTR("super-auth-header"),
			session: &auth_manager.Session{
				DomainId: 1,
				UserId:   3,
				Expire:   time.Now().Add(24 * time.Hour).Unix(),
				RoleIds:  []int{1},
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
			f := func(token string) (*auth_manager.Session, error) {
				if tt.session == nil {
					return nil, fmt.Errorf("empty session")
				}

				tt.session.Id = token
				tt.session.Token = token

				return tt.session, nil
			}

			am := authmock.NewMockManager(t)
			am.EXPECT().GetSession(mock.AnythingOfType("string")).RunAndReturn(f).Maybe()
			if tt.token != nil {
				md := metadata.New(map[string]string{
					hdrTokenAccess: *tt.token,
				})

				ctx = metadata.NewIncomingContext(ctx, md)
			}

			_, err := AuthUnaryServerInterceptor(am)(ctx, nil, info, handler)
			if err != nil {
				if tt.expected.cause != nil {
					assert.ErrorContains(t, werror.Cause(err), tt.expected.cause.Error())
				}

				assert.Equal(t, tt.expected.code, werror.Code(err))
				assert.ErrorIs(t, err, tt.expected.err)
			}
		})
	}
}
