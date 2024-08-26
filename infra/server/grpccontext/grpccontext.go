package grpccontext

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
)

type grpcContextKey struct{}

type GRPCServerContext struct {
	SignedInUser *model.SignedInUser
	RequestId    string
}

func FromContext(ctx context.Context) *GRPCServerContext {
	grpcContext, ok := ctx.Value(grpcContextKey{}).(*GRPCServerContext)
	if !ok {
		return &GRPCServerContext{}
	}

	return grpcContext
}

func SetUser(ctx context.Context, user *model.SignedInUser) context.Context {
	grpcContext := FromContext(ctx)
	if grpcContext == nil {
		grpcContext = &GRPCServerContext{}
	}

	grpcContext.SignedInUser = user

	return context.WithValue(ctx, grpcContextKey{}, grpcContext)
}

func SetRequestId(ctx context.Context, requestId string) context.Context {
	grpcContext := FromContext(ctx)
	if grpcContext == nil {
		grpcContext = &GRPCServerContext{}
	}

	grpcContext.RequestId = requestId

	return context.WithValue(ctx, grpcContextKey{}, grpcContext)
}
