package werror

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

// Wrapper knows how to wrap errors with context information.
type Wrapper interface {

	// Wrap returns a new error, wrapping the argument, and typically adding some context information.
	Wrap(err error) error
}

// WrapperFunc implements Wrapper.
type WrapperFunc func(error) error

// Wrap implements the Wrapper interface.
func (w WrapperFunc) Wrap(err error) error {
	return w(err)
}

// WithValue associates a key/value pair with an error.
func WithValue(key, value interface{}) Wrapper {
	return WrapperFunc(func(err error) error {
		return Set(err, key, value)
	})
}

// AppendMessage a message after the current error message, in the format "original: new".
func AppendMessage(msg string) Wrapper {
	return WrapperFunc(func(err error) error {
		if err == nil {
			return nil
		}

		return Set(err, ErrKeyMessage, err.Error()+": "+msg)
	})
}

// AppendMessagef is the same as AppendMessage, but with a formatted message.
func AppendMessagef(format string, args ...interface{}) Wrapper {
	return WrapperFunc(func(err error) error {
		if err == nil {
			return nil
		}

		return Set(err, ErrKeyMessage, err.Error()+": "+fmt.Sprintf(format, args...))
	})
}

// PrependMessage a message before the current error message, in the format "new: original".
func PrependMessage(msg string) Wrapper {
	return WrapperFunc(func(err error) error {
		if err == nil {
			return nil
		}

		return Set(err, ErrKeyMessage, msg+": "+err.Error())
	})
}

// PrependMessagef is the same as PrependMessage, but with a formatted message.
func PrependMessagef(format string, args ...interface{}) Wrapper {
	return WrapperFunc(func(err error) error {
		if err == nil {
			return nil
		}

		return Set(err, ErrKeyMessage, fmt.Sprintf(format, args...)+": "+err.Error())
	})
}

// WithID set's error ID (pkg.file.func).
func WithID(msg string) Wrapper {
	return WithValue(ErrKeyID, msg)
}

// WithCode associates an gRPC status code with an error.
func WithCode(code codes.Code) Wrapper {
	return WithValue(ErrKeyCode, code)
}

// WithCause sets one error as the cause of another error.
// This is useful for associating errors from lower API levels
// with sentinel errors in higher API levels.
// errors.Is() and errors.As() will traverse both the main chain
// of error wrappers, and down the chain of causes.
func WithCause(err error) Wrapper {
	return WrapperFunc(func(nerr error) error {
		if nerr == nil || err == nil {
			return nerr
		}

		return &errWithCause{err: nerr, cause: err}
	})
}

// Set wraps an error with a key/value pair.
// This is the simplest form of associating a value with an error.
// It is mainly intended as a primitive for writing Wrapper implementations.
func Set(err error, key, value interface{}) error {
	if err == nil {
		return nil
	}

	return &errWithValue{
		err:   err,
		key:   key,
		value: value,
	}
}
