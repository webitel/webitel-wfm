package interceptor

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/webitel/engine/auth_manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pb "github.com/webitel/webitel-wfm/gen/go/api"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const hdrTokenAccess = "X-Webitel-Access"

var reg = regexp.MustCompile(`^(.*\.)`)

// AuthUnaryServerInterceptor returns a server interceptor function to authenticate && authorize unary RPC.
func AuthUnaryServerInterceptor(authcli auth_manager.AuthManager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		token, err := tokenFromContext(ctx)
		if err != nil {
			return nil, werror.NewAuthInvalidTokenError("interceptor.auth.metadata", err)
		}

		session, err := validateSession(authcli, token)
		if err != nil {
			return nil, werror.NewAuthInvalidSessionError("interceptor.auth.session", token, err)
		}

		objClass, action := objClassWithAction(info)
		ok, useRBAC := validateSessionPermission(session, objClass, action)
		if ok { // FIXME: must be !ok
			return nil, werror.NewAuthForbiddenError("interceptor.auth.permission", action.Name(), objClass)
		}

		s := &model.SignedInUser{
			Token:    session.Id,
			DomainId: session.DomainId,
			Id:       session.UserId,
			Object:   objClass,
			UseRBAC:  useRBAC,
			RbacOptions: model.RbacOptions{
				Groups: session.GetAclRoles(),
				Access: action.Value(),
			},
		}

		return handler(grpccontext.SetUser(ctx, s), req)
	}
}

func tokenFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("empty metadata")
	}

	token := md.Get(hdrTokenAccess)
	if len(token) < 1 {
		return "", fmt.Errorf("authorization token is null")
	}

	if token[0] == "" {
		return "", fmt.Errorf("authorization token is empty")
	}

	return token[0], nil
}

func validateSession(authcli auth_manager.AuthManager, token string) (*auth_manager.Session, error) {
	session, err := authcli.GetSession(token)
	if err != nil {
		return nil, fmt.Errorf("client: %s", err)
	}

	if err := session.IsValid(); err != nil {
		return nil, err
	}

	if session.IsExpired() {
		return nil, fmt.Errorf("authorization token is expired")
	}

	return session, nil
}

func objClassWithAction(info *grpc.UnaryServerInfo) (string, auth_manager.PermissionAccess) {
	service, method := splitFullMethodName(info.FullMethod)
	objClass := pb.WebitelAPI[service].ObjClass
	action := pb.WebitelAPI[service].WebitelMethods[method].Access

	return objClass, auth_manager.PermissionAccess(action)
}

func validateSessionPermission(session *auth_manager.Session, objClass string, action auth_manager.PermissionAccess) (bool, bool) {
	permission := session.GetPermission(objClass)
	switch action {
	case auth_manager.PERMISSION_ACCESS_CREATE:
		if !permission.CanCreate() {
			return false, false
		}
	case auth_manager.PERMISSION_ACCESS_READ:
		if !permission.CanRead() {
			return false, false
		}
	case auth_manager.PERMISSION_ACCESS_UPDATE:
		if !permission.CanRead() && !permission.CanUpdate() {
			return false, false
		}
	case auth_manager.PERMISSION_ACCESS_DELETE:
		if !permission.CanDelete() {
			return false, false
		}
	default:
		return false, false
	}

	if session.UseRBAC(action, permission) {
		return true, true
	}

	return true, false
}

func splitFullMethodName(fullMethod string) (string, string) {
	fullMethod = strings.TrimPrefix(fullMethod, "/") // remove leading slash
	if i := strings.Index(fullMethod, "/"); i >= 0 {
		return reg.ReplaceAllString(fullMethod[:i], ""), fullMethod[i+1:]
	}

	return "unknown", "unknown"
}
