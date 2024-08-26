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
	"google.golang.org/grpc/metadata"

	authmock "github.com/webitel/webitel-wfm/gen/go/mocks/github.com/webitel/engine/auth_manager"
)

func TestAuthUnaryServerInterceptor(t *testing.T) {
	am := authmock.NewMockAuthManager(t)
	interceptor := AuthUnaryServerInterceptor(am)
	/*	auth.DefaultObjClassServiceName = auth.ObjClass{
		"FakeService": auth.ObjClassService{
			ObjClass: "fake",
		},
	}*/

	handler := func(context.Context, any) (any, error) {
		return "good", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/FakeService/FakeMethod",
	}

	tests := map[string]struct {
		token    []string
		session  *auth_manager.Session
		expected string
	}{
		"success": {
			token: []string{"super-auth-header"},
			session: &auth_manager.Session{
				DomainId: 1,
				UserId:   3,
				Expire:   time.Now().Add(24 * time.Hour).Unix(),
				RoleIds:  []int{1},
				Scopes: []auth_manager.SessionPermission{{
					Name:   "fake",
					Access: auth_manager.PERMISSION_ACCESS_CREATE.Value(),
					Obac:   true,
				}},
			},
		},
		"empty metadata": {
			token:   nil,
			session: &auth_manager.Session{},
		},
		"authorization token is empty": {
			token:   []string{"", ""},
			session: &auth_manager.Session{},
		},
		"empty session": {
			token:   []string{"super-auth-header"},
			session: nil,
		},
		"session is invalid": {
			token: []string{"super-auth-header"},
			session: &auth_manager.Session{
				DomainId: 0,
				UserId:   0,
				Expire:   time.Now().Add(1 * time.Hour).Unix(),
				RoleIds:  []int{},
			},
		},
		"authorization token is expired": {
			token: []string{"super-auth-header"},
			session: &auth_manager.Session{
				DomainId: 1,
				UserId:   3,
				Expire:   time.Now().Truncate(1 * time.Hour).Unix(),
				RoleIds:  []int{1},
			},
		},
		"forbidden": {
			token: []string{"super-auth-header"},
			session: &auth_manager.Session{
				DomainId: 1,
				UserId:   3,
				Expire:   time.Now().Add(24 * time.Hour).Unix(),
				RoleIds:  []int{1},
			},
		},
	}

	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			f := func(token string) (*auth_manager.Session, error) {
				if tt.session == nil {
					return nil, fmt.Errorf("nil session")
				}

				tt.session.Id = token
				tt.session.Token = token

				return tt.session, nil
			}

			am.EXPECT().GetSession(mock.AnythingOfType("string")).RunAndReturn(f).Maybe()
			if len(tt.token) > 0 {
				md := metadata.New(map[string]string{})
				for _, v := range tt.token {
					md.Set(hdrTokenAccess, v)
				}

				ctx = metadata.NewIncomingContext(ctx, md)
			}

			_, err := interceptor(ctx, nil, info, handler)
			if err != nil {
				assert.ErrorContains(t, err, scenario)
			}
		})
	}
}
