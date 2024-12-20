package werror_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

func TestNew(t *testing.T) {
	err := werror.New("bang")
	assert.EqualError(t, err, "bang")

	// werror.New accepts werror.Wrapper options
	err = werror.New("boom", werror.WithValue("raw", "blue"))
	assert.EqualError(t, err, "boom")
	assert.Equal(t, "blue", werror.Value(err, "raw"))
}

func TestWrap(t *testing.T) {
	// capture a stack
	ogerr := errors.New("boom")
	err := werror.Wrap(ogerr)

	// werror.New error should werror.Wrap the old error
	assert.True(t, errors.Is(err, ogerr))

	// werror.Wrap accepts werror.Wrapper args
	err = werror.Wrap(err, werror.WithValue("raw", "hi"), werror.WithCode(6))
	assert.Equal(t, "hi", werror.Value(err, "raw"))
	assert.Equal(t, codes.Code(6), werror.Code(err))

	// werror.Wrapping nil -> nil
	assert.Nil(t, werror.Wrap(nil))
}

func TestAppend(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Append(nil, "big"))

	// append message
	assert.EqualError(t, werror.Append(werror.New("blue"), "big"), "blue: big")

	// werror.Wrapper varargs
	err := werror.Append(werror.New("blue"), "big", werror.WithCode(3))
	assert.Equal(t, codes.Code(3), werror.Code(err))
	assert.EqualError(t, err, "blue: big")
}

func TestAppendf(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Appendf(nil, "big %s", "red"))

	// append message
	assert.EqualError(t, werror.Appendf(werror.New("blue"), "big %s", "red"), "blue: big red")

	// werror.Wrapper varargs
	err := werror.Appendf(werror.New("blue"), "big %s", werror.WithCode(3), "red")
	assert.Equal(t, codes.Code(3), werror.Code(err))
	assert.EqualError(t, err, "blue: big red")
}

func TestPrepend(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Prepend(nil, "big"))

	// append message
	assert.EqualError(t, werror.Prepend(werror.New("blue"), "big"), "big: blue")

	// werror.Wrapper varargs
	err := werror.Prepend(werror.New("blue"), "big", werror.WithCode(3))
	assert.Equal(t, codes.Code(3), werror.Code(err))
	assert.EqualError(t, err, "big: blue")
}

func TestPrependf(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Prependf(nil, "big %s", "red"))

	// append message
	assert.EqualError(t, werror.Prependf(werror.New("blue"), "big %s", "red"), "big red: blue")

	// werror.Wrapper varargs
	err := werror.Prependf(werror.New("blue"), "big %s", werror.WithCode(3), "red")
	assert.Equal(t, codes.Code(3), werror.Code(err))
	assert.EqualError(t, err, "big red: blue")
}

func TestValue(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Value(nil, "color"))

	err := werror.New("bang")
	assert.Nil(t, werror.Value(err, "color"))

	err = werror.Wrap(err, werror.WithValue("color", "red"))
	assert.Equal(t, "red", werror.Value(err, "color"))

	// will traverse non-werror in the chain
	err = werror.New("bam", werror.WithValue("color", "red"))
	err = &UnwrapperError{err}
	err = werror.Wrap(err, werror.WithValue("raw", "yikes"))
	assert.Equal(t, "red", werror.Value(err, "color"))

	// will not search the cause chain
	err = werror.New("whoops", werror.WithCause(werror.New("yikes", werror.WithValue("color", "red"))))
	assert.Nil(t, werror.Value(err, "color"))

	// if the current error and the cause both have a werror.Value for the
	// same key, the top errors werror.Value will always take precedence, even
	// if the cause was attached to the error after the werror.Value was.

	err = werror.New("boom", werror.WithValue("color", "red"))
	err = werror.Wrap(err, werror.WithCause(werror.New("io error", werror.WithValue("color", "blue"))))
	assert.Equal(t, "red", werror.Value(err, "color"))
}

func TestValues(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Values(nil))

	// error with no values should still be nil
	assert.Nil(t, werror.Values(errors.New("boom")))

	// create an error chain with a few values attached, and a non-werror
	// in the middle.
	err := werror.New("boom", werror.WithValue("raw", "bam"), werror.WithCode(4))
	err = &UnwrapperError{err}
	err = werror.Wrap(err, werror.WithValue("color", "red"))

	values := werror.Values(err)

	assert.Equal(t, map[interface{}]interface{}{
		"raw":             "bam",
		werror.ErrKeyCode: codes.Code(4),
		"color":           "red",
	}, values)
}

func BenchmarkValues(b *testing.B) {
	// create an error chain with a few values attached
	// and a non-werror in the middle
	err := werror.New("boom", werror.WithValue("raw", "bam"), werror.WithCode(4))
	err = &UnwrapperError{err}
	err = werror.Wrap(err, werror.WithValue("color", "red"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		werror.Values(err)
	}
}

func TestCode(t *testing.T) {
	// nil -> 200
	assert.Equal(t, codes.Code(200), werror.Code(nil))

	// default to 500
	assert.Equal(t, codes.Code(500), werror.Code(errors.New("boom")))

	// set with werror.Wrapper
	assert.Equal(t, codes.Code(404), werror.Code(werror.New("boom", werror.WithCode(404))))

	// works when werror.Value is deep in stack
	err := werror.New("bam", werror.WithCode(404))
	err = &UnwrapperError{err}
	err = werror.Wrap(err, werror.WithValue("raw", "yikes"))
	assert.Equal(t, codes.Code(404), werror.Code(err))
}

func TestCause(t *testing.T) {
	// nil -> nil
	assert.Nil(t, werror.Cause(nil))

	// no cause -> nil
	assert.Nil(t, werror.Cause(werror.New("boom")))

	// with cause
	root := errors.New("boom")
	err := werror.New("yikes", werror.WithCause(root))
	assert.EqualError(t, werror.Cause(err), "boom")

	// with nil cause, should be no-op
	err = werror.New("yikes", werror.WithCause(nil))
	assert.Nil(t, werror.Cause(err))
}
