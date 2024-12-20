package werror_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

func TestErrWithValue_Format(t *testing.T) {
	// %v and %s print the same as err.Error() if there is no cause
	err := werror.Wrap(errors.New("Hi"), werror.WithValue("test", "value"))
	assert.Equal(t, fmt.Sprintf("%v", err), err.Error())
	assert.Equal(t, fmt.Sprintf("%s", err), err.Error())

	// %q returns err.Error() as a golang literal
	assert.Equal(t, fmt.Sprintf("%q", err), fmt.Sprintf("%q", err.Error()))

	// if there is a cause in the chain, include it.
	err = werror.Wrap(err, werror.WithCause(werror.New("Bye")), werror.WithValue("color", "red"))
	assert.Equal(t, fmt.Sprintf("%v", err), "Hi: Bye")
	assert.Equal(t, fmt.Sprintf("%s", err), "Hi: Bye")

	err = werror.Wrap(err, werror.WithValue("raw", "blue"))
	assert.Equal(t, fmt.Sprintf("%+v", err), werror.Details(err))
}

func TestErrWithCause_Format(t *testing.T) {
	// %v and %s also print the cause, if there is one
	err := werror.New("Hi", werror.WithCause(werror.New("Bye")))
	assert.Equal(t, fmt.Sprintf("%v", err), "Hi: Bye")
	assert.Equal(t, fmt.Sprintf("%s", err), "Hi: Bye")

	// %q returns err.Error() as a golang literal
	assert.Equal(t, fmt.Sprintf("%q", err), fmt.Sprintf("%q", err.Error()))

	// %+v should return full details, including properties registered with RegisterXXX() functions
	// and the stack.
	assert.Equal(t, fmt.Sprintf("%+v", err), werror.Details(err))
}

func TestErrWithValue_Error(t *testing.T) {
	err := werror.New("red")
	assert.Equal(t, "red", err.Error())

	err = werror.Wrap(err, werror.WithValue("raw", "blue"))
	assert.Equal(t, "blue", werror.Value(err, "raw"))
}

func TestErrWithCause_Error(t *testing.T) {
	err := werror.New("blue", werror.WithCause(werror.New("red")))
	assert.Equal(t, "blue", err.Error())
}

// UnwrapperError is a simple error implementation that wraps another error, and implements `Unwrap() error`.
// It is used to test when errors not created by this package are inserted in the chain of wrapped errors.
type UnwrapperError struct {
	err error
}

func (w *UnwrapperError) Error() string {
	return w.err.Error()
}

func (w *UnwrapperError) Unwrap() error {
	return w.err
}

func TestErrWithValue_Unwrap(t *testing.T) {
	e1 := werror.New("blue", werror.WithValue("color", "red"))
	assert.EqualError(t, errors.Unwrap(e1), "blue")
}

func TestErrWithCause_Unwrap(t *testing.T) {
	err := werror.Wrap(errors.New("blue"), werror.WithCause(errors.New("red")))

	// unwrapping the layers should return blue, then the cause (red).
	// first unwrap should return blue
	var layers []string
	var unwrapped error = err
	for unwrapped != nil {
		layers = append(layers, unwrapped.Error())
		unwrapped = errors.Unwrap(unwrapped)
	}

	assert.Equal(t, []string{"blue", "red"}, layers)

	// If another cause is attached, it should override the older cause.
	// Unwrap should no longer return the older cause.
	unwrapped = werror.Wrap(err, werror.WithCause(errors.New("yellow")))
	layers = nil
	for unwrapped != nil {
		layers = append(layers, unwrapped.Error())
		unwrapped = errors.Unwrap(unwrapped)
	}

	assert.Equal(t, []string{"blue", "yellow"}, layers)
}

func TestErrWithValue_String(t *testing.T) {
	err := werror.New("blue")
	assert.Equal(t, "blue", err.Error())

	err = werror.Wrap(err, werror.WithValue("raw", "red"))
	assert.Equal(t, "red", werror.Value(err, "raw"))
}

func TestErrWithCause_String(t *testing.T) {
	assert.Equal(t, "blue", werror.New("blue", werror.WithCause(werror.New("green"))).Error())
}

func TestIs(t *testing.T) {
	// an error is all the errors it wraps
	e1 := werror.New("blue")
	e2 := werror.Wrap(e1, werror.WithCode(5))
	assert.True(t, errors.Is(e2, e1))
	assert.False(t, errors.Is(e1, e2))

	// is works through other unwrapper implementations
	e3 := &UnwrapperError{err: e2}
	e4 := werror.Wrap(e3, werror.WithValue("raw", "hi"))
	assert.True(t, errors.Is(e4, e3))
	assert.True(t, errors.Is(e4, e2))
	assert.True(t, errors.Is(e4, e1))

	// an error is also any of the causes
	rootCause := errors.New("ioerror")
	rootCause1 := werror.Wrap(rootCause)
	outererr := werror.New("failed", werror.WithCause(rootCause1))
	outererr1 := werror.Wrap(outererr, werror.WithValue("raw", "sorry!"))

	assert.True(t, errors.Is(outererr1, outererr))
	assert.True(t, errors.Is(outererr1, rootCause1))
	assert.True(t, errors.Is(outererr1, rootCause))

	// but only the latest cause
	newCause := errors.New("new cause")
	outererr1 = werror.Wrap(outererr1, werror.WithCause(newCause))
	assert.ErrorIs(t, outererr1, newCause)
	assert.NotErrorIs(t, outererr1, rootCause)
	assert.NotErrorIs(t, outererr1, rootCause1)
}

type redError int

func (*redError) Error() string {
	return "red error"
}

func TestAs(t *testing.T) {
	err := werror.New("blue error")

	// as will find matching errors in the chain
	var rerr *redError
	assert.False(t, errors.As(err, &rerr))
	assert.Nil(t, rerr)

	rr := redError(3)
	err = werror.Wrap(&rr)

	assert.True(t, errors.As(err, &rerr))
	assert.Equal(t, &rr, rerr)

	rerr = nil

	// test that it works with non-werror in the chain
	err = &UnwrapperError{err: err}
	assert.True(t, errors.As(err, &rerr))
	assert.Equal(t, &rr, rerr)

	err = werror.Wrap(err, werror.PrependMessage("asdf"))

	rerr = nil

	assert.True(t, errors.As(err, &rerr))
	assert.Equal(t, &rr, rerr)

	// will search causes as well
	err = werror.New("boom", werror.WithCause(err))

	rerr = nil

	assert.True(t, errors.As(err, &rerr))
	assert.Equal(t, &rr, rerr)

	// but only the latest cause
	err = werror.Wrap(err, werror.WithCause(errors.New("new cause")))
	assert.False(t, errors.As(err, &rerr))
}
