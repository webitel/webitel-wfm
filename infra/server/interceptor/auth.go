package interceptor

import (
	"context"
	"regexp"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/infra/webitel/auth"
	"github.com/webitel/webitel-wfm/internal/model"
)

const hdrTokenAccess = "X-Webitel-Access"

var reg = regexp.MustCompile(`^(.*\.)`)

var (
	ErrInvalidToken    = errors.Unauthenticated("auth token is invalid", errors.WithID("interceptor.auth.metadata"))
	ErrInvalidSession  = errors.Unauthenticated("auth session is invalid", errors.WithID("interceptor.auth.session"))
	ErrLicenseRequired = errors.Forbidden("license required", errors.WithID("interceptor.auth.license"))
	ErrForbidden       = errors.Forbidden("permission denied on resource (or it might not exist)", errors.WithID("interceptor.auth.permission"))
)

// AuthUnaryServerInterceptor returns a server interceptor function to authenticate && authorize unary RPC.
func AuthUnaryServerInterceptor(authcli auth.Manager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		token, err := tokenFromContext(ctx)
		if err != nil {
			return nil, errors.Wrap(ErrInvalidToken, errors.WithCause(err))
		}

		session, err := validateSession(ctx, authcli, token)
		if err != nil {
			return nil, errors.Wrap(ErrInvalidSession, errors.WithCause(err))
		}

		objClass, licenses, action := objClassWithAction(info)
		if len(licenses) > 0 {
			nfl := make([]string, 0, len(licenses)) // not found licenses
			for _, license := range licenses {
				if !session.HasLicense(license) {
					nfl = append(nfl, license)
				}
			}

			if len(nfl) > 0 {
				return nil, errors.Wrap(ErrLicenseRequired, errors.WithValue("objclass", objClass),
					errors.WithValue("license", strings.Join(nfl, ", ")),
				)
			}
		}

		ok, useRBAC := validateSessionPermission(session, objClass, action)
		_ = ok
		//if ok { // FIXME: must be !ok
		//	return nil, errors.Wrap(ErrForbidden, errors.WithValue("objclass", objClass), errors.WithValue("action", action.Name()))
		//}

		s := &model.SignedInUser{
			Token:    session.Token,
			DomainId: session.DomainID,
			Id:       session.UserID,
			Object:   objClass,
			UseRBAC:  useRBAC,
			RbacOptions: model.RbacOptions{
				Groups: session.Roles(),
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
		return "", errors.New("can't find authorization token")
	}

	if token[0] == "" {
		return "", errors.New("empty authorization token")
	}

	return token[0], nil
}

func validateSession(ctx context.Context, authcli auth.Manager, token string) (*auth.Session, error) {
	session, err := authcli.Session(ctx, token)
	if err != nil {
		return nil, errors.Prepend(err, "client")
	}

	if session == nil {
		return nil, errors.New("empty session")
	}

	if err := session.Validate(); err != nil {
		return nil, err
	}

	if session.Expired() {
		return nil, errors.New("expired authorization token")
	}

	return session, nil
}

func objClassWithAction(info *grpc.UnaryServerInfo) (string, []string, auth.Access) {
	service, method := splitFullMethodName(info.FullMethod)
	objClass := pb.WebitelAPI[service].ObjClass
	licenses := pb.WebitelAPI[service].AdditionalLicenses
	action := pb.WebitelAPI[service].WebitelMethods[method].Access

	// TODO: make licenses unique list
	return objClass, append(licenses, "WFM"), auth.Access(action)
}

func validateSessionPermission(session *auth.Session, objClass string, action auth.Access) (bool, bool) {
	permission := session.Permission(objClass)
	switch action {
	case auth.AccessCreate:
		if !permission.CanCreate() {
			return false, false
		}
	case auth.AccessRead:
		if !permission.CanRead() {
			return false, false
		}
	case auth.AccessUpdate:
		if !permission.CanRead() && !permission.CanUpdate() {
			return false, false
		}
	case auth.AccessDelete:
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
