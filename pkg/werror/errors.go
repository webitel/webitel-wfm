package werror

import (
	"errors"
	"fmt"
	"reflect"
)

// Is reports whether any error in err's chain matches target.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
func Is(err, target error) bool { return errors.Is(err, target) }

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error matches target if the error's concrete value is assignable to the value
// pointed to by target, or if the error has a method As(interface{}) bool such that
// As(target) returns true. In the latter case, the As method is responsible for
// setting target.
//
// As will panic if target is not a non-nil pointer to either a type that implements
// error, or to any interface type. As returns false if err is nil.
func As(err error, target interface{}) bool { return errors.As(err, target) }

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// Otherwise, Unwrap returns nil.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

type ErrKey int

const (
	ErrKeyNone ErrKey = iota
	ErrKeyID
	ErrKeyMessage
	ErrKeyCode
)

func (e ErrKey) String() string {
	return []string{"none", "id", "message", "code"}[e]
}

// formatError adds a Format implementation to an error.
type formatError struct {
	error
}

// Format implements fmt.Formatter
func (e *formatError) Format(s fmt.State, verb rune) {
	Format(s, verb, e)
}

// String implements fmt.Stringer
func (e *formatError) String() string {
	return e.Error()
}

// Unwrap returns the next wrapped error.
func (e *formatError) Unwrap() error {
	return e.error
}

type errWithValue struct {
	err        error
	key, value interface{}
}

// Format implements fmt.Formatter
func (e *errWithValue) Format(s fmt.State, verb rune) {
	Format(s, verb, e)
}

// Error implements golang's error interface
// returns the message value if set, otherwise
// delegates to inner error
func (e *errWithValue) Error() string {
	if e.key == ErrKeyMessage {
		if s, ok := e.value.(string); ok {
			return s
		}
	}
	return e.err.Error()
}

// String implements fmt.Stringer
func (e *errWithValue) String() string {
	return e.Error()
}

// Unwrap returns the next wrapped error.
func (e *errWithValue) Unwrap() error {
	return e.err
}

func (e *errWithValue) isWError() {}

type errWithCause struct {
	err   error
	cause error
}

func (e *errWithCause) Unwrap() error {
	// skip through any directly nested errWithCauses.
	// our implementation of Is/As already recursed through them,
	// so we want to dig down to the first non-errWithCause.

	nextErr := e.err
	for {
		if e, ok := nextErr.(*errWithCause); ok {
			nextErr = e.err
		} else {
			break
		}
	}

	// errWithCause.Is/As() also already checked nextErr, so we want to
	// unwrap it and get to the next error down.
	nextErr = errors.Unwrap(nextErr)

	// We've reached the end of this wrapper chain.
	if nextErr == nil {
		return e.cause
	}

	// Return a new errWithCause wrapper, wrapping next error, but bundling
	// it will our cause, ignoring the causes of the errWithCauses we skip
	// over above.
	// This is how we carry the latest cause along as we unwrap the chain.
	// When we get to the end of the chain, we'll return this latest cause.
	return &errWithCause{err: nextErr, cause: e.cause}
}

func (e *errWithCause) String() string {
	return e.Error()
}

func (e *errWithCause) Error() string {
	return e.err.Error()
}

func (e *errWithCause) Format(f fmt.State, verb rune) {
	Format(f, verb, e)
}

func (e *errWithCause) Is(target error) bool {
	// This does most of what errors.Is() does, by delegating
	// to the nested error.
	// But it does not use Unwrap to recurse any further.
	// This just compares target with next error in the stack.
	isComparable := reflect.TypeOf(target).Comparable()
	if isComparable && e.err == target {
		return true
	}

	// Since errWithCause implements Is(),
	// this will effectively recurse through any directly nested errWithCauses.
	if x, ok := e.err.(interface{ Is(error) bool }); ok && x.Is(target) {
		return true
	}

	return false
}

func (e *errWithCause) As(target interface{}) bool {
	// This does most of what errors.As() does, by delegating
	// to the nested error. But it does not use Unwrap to recurse
	// any further. This just compares target with next error in the stack.
	val := reflect.ValueOf(target)
	typ := val.Type()
	targetType := typ.Elem()
	if reflect.TypeOf(e.err).AssignableTo(targetType) {
		val.Elem().Set(reflect.ValueOf(e.err))

		return true
	}

	// Since errWithCause implements As(),
	// this will effectively recurse through any directly nested errWithCauses.
	if x, ok := e.err.(interface{ As(interface{}) bool }); ok && x.As(target) {
		return true
	}

	return false
}

func (e *errWithCause) isWError() {}
