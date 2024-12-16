// Package werror provides a way to return detailed information
// for an RPC request error. The error is normally JSON encoded.
package werror

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
)

// один тип загальний з полями ід, мессадж, статус, інфо
// ід, мессадж, статус як константа
// метод для сеттінгу деталей map[string]any

// New creates a new error, with a stack attached.
// The equivalent of golang's errors.New()
func New(msg string, wrappers ...Wrapper) error {
	return Wrap(errors.New(msg), wrappers...)
}

func Forbidden(msg string, wrappers ...Wrapper) error {
	return New(msg, append(wrappers, WithCode(codes.PermissionDenied))...)
}

func Unauthenticated(msg string, wrappers ...Wrapper) error {
	return New(msg, append(wrappers, WithCode(codes.Unauthenticated))...)
}

func NotFound(msg string, wrappers ...Wrapper) error {
	return New(msg, append(wrappers, WithCode(codes.NotFound))...)
}

func InvalidArgument(msg string, wrappers ...Wrapper) error {
	return New(msg, append(wrappers, WithCode(codes.InvalidArgument))...)
}

func Aborted(msg string, wrappers ...Wrapper) error {
	return New(msg, append(wrappers, WithCode(codes.Aborted))...)
}

func Internal(msg string, wrappers ...Wrapper) error {
	return New(msg, append(wrappers, WithCode(codes.Internal))...)
}

// Wrap adds context to errors by applying Wrappers.
// See WithXXX() functions for Wrappers supplied by this package.
func Wrap(err error, wrappers ...Wrapper) error {
	if err == nil {
		return nil
	}

	for _, w := range wrappers {
		err = w.Wrap(err)
	}

	// Ensure the resulting error implements Formatter.
	if _, ok := err.(fmt.Formatter); !ok {
		err = &formatError{err}
	}

	return err
}

// Prepend is a convenience function for the PrependMessage wrapper.
func Prepend(err error, msg string, wrappers ...Wrapper) error {
	return Wrap(err, append(wrappers, PrependMessage(msg))...)
}

// Prependf is a convenience function for the PrependMessagef wrapper.
// The args can be format arguments mixed with Wrappers.
func Prependf(err error, format string, args ...interface{}) error {
	fmtArgs, wrappers := splitWrappers(args)

	return Wrap(err, append(wrappers, PrependMessagef(format, fmtArgs...))...)
}

// Append is a convenience function for the AppendMessage wrapper.
func Append(err error, msg string, wrappers ...Wrapper) error {
	return Wrap(err, append(wrappers, AppendMessage(msg))...)
}

// Appendf is a convenience function for the AppendMessagef wrapper.
// The args can be format arguments mixed with Wrappers.
func Appendf(err error, format string, args ...interface{}) error {
	fmtArgs, wrappers := splitWrappers(args)

	return Wrap(err, append(wrappers, AppendMessagef(format, fmtArgs...))...)
}

func ID(err error) string {
	if err == nil {
		return ""
	}

	if id, ok := Value(err, ErrKeyID).(string); ok {
		return id
	}

	return ""
}

// Code converts an error to gRPC status code.
// All errors map to Internal, unless the error has code attached.
// If err is nil, returns OK.
func Code(err error) codes.Code {
	if err == nil {
		return 200
	}

	code, _ := Value(err, ErrKeyCode).(codes.Code)
	if code == 0 {
		return 500
	}

	return code
}

// Cause returns the cause of the argument.
func Cause(err error) error {
	var causer *errWithCause
	if errors.As(err, &causer) {
		return causer.cause
	}

	return nil
}

// Value returns the value for key, or nil if not set.
// Will not search causes.
func Value(err error, key interface{}) interface{} {
	v, ok := Lookup(err, key)
	if !ok {
		return nil
	}

	return v
}

// Lookup returns the value for the key, and a boolean indicating
// whether the value was set.
// Will not search causes.
func Lookup(err error, key interface{}) (interface{}, bool) {
	var werr interface {
		error
		isWError()
	}

	// I've tried implementing this logic a few different ways.  It's tricky:
	//
	// - Lookup should only search the current error, but not causes.
	//   errWithCause's Unwrap() will eventually unwrap to the cause,
	//   so we don't want to just search the entire stream of errors
	//   returned by Unwrap.
	// - We need to handle cases where error implementations created outside
	//   this package are in the middle of the chain.  We need to use Unwrap
	//   in these cases to traverse those errors and dig down to the next
	//   werror.
	// - Some error packages, including our own, do funky stuff with Unwrap(),
	//   returning shim types to control the unwrapping order, rather than
	//   the actual, raw wrapped error. Typically, these shims implement
	//   Is/As to delegate to the raw error they encapsulate, but implement
	//   Unwrap by encapsulating the raw error in another shim. So if we're looking
	//   for a raw error type, we can't just use Unwrap() and do type assertions
	//   against the result. We have to use errors.As(), to allow the shims to delegate
	//   the type assertion to the raw error correctly.
	//
	// Based on all these constraints, we use errors.As() with an internal interface
	// that can only be implemented by our internal error types.  When one is found,
	// we handle each of our internal types as a special case. For errWithCause, we
	// traverse to the wrapped error, ignoring the cause and the funky Unwrap logic.
	// We could have just used errors.As(err, *errWithValue), but that would have
	// traversed into the causes.

	for {
		switch t := err.(type) {
		case *errWithValue:
			if t.key == key {
				return t.value, true
			}

			err = t.err
		case *errWithCause:
			err = t.err
		default:
			if errors.As(err, &werr) {
				err = werr
			} else {
				return nil, false
			}
		}
	}
}

// Values returns a map of all values attached to the error
// If a key has been attached multiple times, the map will
// contain the last value mapped.
func Values(err error) map[interface{}]interface{} {
	var values map[interface{}]interface{}
	for err != nil {
		if e, ok := err.(*errWithValue); ok {
			if _, ok := values[e.key]; !ok {
				if values == nil {
					values = map[interface{}]interface{}{}
				}

				values[e.key] = e.value
			}
		}

		err = errors.Unwrap(err)
	}

	return values
}

func splitWrappers(args []interface{}) ([]interface{}, []Wrapper) {
	var wrappers []Wrapper

	// pull out the args which are wrappers
	n := 0
	for _, arg := range args {
		if w, ok := arg.(Wrapper); ok {
			wrappers = append(wrappers, w)
		} else {
			args[n] = arg
			n++
		}
	}
	args = args[:n]

	return args, wrappers
}
