package interceptor

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/webitel/webitel-go-kit/logging/wlog"
	"google.golang.org/grpc"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

var ErrPanicReceived = werror.New("panic occurred, please contact our support", werror.WithID("interceptor.panic"))

// panicError is an error that is used to recover from a panic.
type panicError struct {
	panic any
	stack []byte
}

// Error implements the error interface.
func (e *panicError) Error() string {
	return fmt.Sprintf("panic caught: %v\n\n%s", e.panic, e.stack)
}

// RecoveryHandlerFuncContext is a function that recovers from the panic `p` by returning an `error`.
// The context can be used to extract request scoped metadata and context values.
type recoveryHandlerFuncContext func(ctx context.Context, p any) (err error)

// RecoveryUnaryServerInterceptor returns a new unary server interceptor for panic recovery.
func RecoveryUnaryServerInterceptor(log *wlog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = recoverFrom(ctx, r, grpcPanicRecoveryHandler(log))
			}
		}()

		return handler(ctx, req)
	}
}

func recoverFrom(ctx context.Context, p any, r recoveryHandlerFuncContext) error {
	if r != nil {
		return r(ctx, p)
	}

	stack := make([]byte, 64<<10)
	stack = stack[:runtime.Stack(stack, false)]

	return &panicError{panic: p, stack: stack}
}

func grpcPanicRecoveryHandler(log *wlog.Logger) func(context.Context, any) error {
	return func(ctx context.Context, p any) (err error) {
		log.Error(fmt.Sprintf("recovered from panic: %s", debug.Stack()))

		return werror.Wrap(ErrPanicReceived, werror.WithValue("stack", p))
	}
}
