// Package werror provides a way to return detailed information
// for an RPC request error. The error is normally JSON encoded.
package werror

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
)

type RPCError struct {
	Id      string     `json:"id"`
	Code    int        `json:"code"`
	Detail  string     `json:"detail"`
	Status  string     `json:"status"`
	RPCCode codes.Code `json:"-"`
}

func NewRPCError(id string, code codes.Code, msg string) *RPCError {
	e := &RPCError{
		Id:      id,
		Code:    HTTPStatusFromCode(code),
		Detail:  msg,
		Status:  http.StatusText(int(code)),
		RPCCode: code,
	}

	e.Status = http.StatusText(e.Code)

	return e
}

func (e *RPCError) JSON() string {
	b, _ := json.Marshal(e)

	return string(b)
}

func (e *RPCError) Error() string {
	return e.JSON()
}

// HTTPStatusFromCode converts a gRPC error code into the corresponding HTTP response status.
// See: https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
func HTTPStatusFromCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		// Note, this deliberately doesn't translate to the similarly named '412 Precondition Failed' HTTP response status.
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	}

	return http.StatusInternalServerError
}
