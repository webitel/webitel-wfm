package werror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestNew(t *testing.T) {
	err := New("bang")
	assert.EqualError(t, err, "bang")

	// New accepts wrapper options
	err = New("boom", WithValue("raw", "blue"))
	assert.EqualError(t, err, "boom")
	assert.Equal(t, "blue", Value(err, "raw"))
}

func TestWrap(t *testing.T) {
	// capture a stack
	ogerr := errors.New("boom")
	err := Wrap(ogerr)

	// new error should wrap the old error
	assert.True(t, errors.Is(err, ogerr))

	// wrap accepts wrapper args
	err = Wrap(err, WithValue("raw", "hi"), WithCode(6))
	assert.Equal(t, "hi", Value(err, "raw"))
	assert.Equal(t, codes.Code(6), Code(err))

	// wrapping nil -> nil
	assert.Nil(t, Wrap(nil))
}

func TestAppend(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Append(nil, "big"))

	// append message
	assert.EqualError(t, Append(New("blue"), "big"), "blue: big")

	// wrapper varargs
	err := Append(New("blue"), "big", WithCode(3))
	assert.Equal(t, codes.Code(3), Code(err))
	assert.EqualError(t, err, "blue: big")
}

func TestAppendf(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Appendf(nil, "big %s", "red"))

	// append message
	assert.EqualError(t, Appendf(New("blue"), "big %s", "red"), "blue: big red")

	// wrapper varargs
	err := Appendf(New("blue"), "big %s", WithCode(3), "red")
	assert.Equal(t, codes.Code(3), Code(err))
	assert.EqualError(t, err, "blue: big red")
}

func TestPrepend(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Prepend(nil, "big"))

	// append message
	assert.EqualError(t, Prepend(New("blue"), "big"), "big: blue")

	// wrapper varargs
	err := Prepend(New("blue"), "big", WithCode(3))
	assert.Equal(t, codes.Code(3), Code(err))
	assert.EqualError(t, err, "big: blue")
}

func TestPrependf(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Prependf(nil, "big %s", "red"))

	// append message
	assert.EqualError(t, Prependf(New("blue"), "big %s", "red"), "big red: blue")

	// wrapper varargs
	err := Prependf(New("blue"), "big %s", WithCode(3), "red")
	assert.Equal(t, codes.Code(3), Code(err))
	assert.EqualError(t, err, "big red: blue")
}

func TestValue(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Value(nil, "color"))

	err := New("bang")
	assert.Nil(t, Value(err, "color"))

	err = Wrap(err, WithValue("color", "red"))
	assert.Equal(t, "red", Value(err, "color"))

	// will traverse non-werror in the chain
	err = New("bam", WithValue("color", "red"))
	err = &UnwrapperError{err}
	err = Wrap(err, WithValue("raw", "yikes"))
	assert.Equal(t, "red", Value(err, "color"))

	// will not search the cause chain
	err = New("whoops", WithCause(New("yikes", WithValue("color", "red"))))
	assert.Nil(t, Value(err, "color"))

	// if the current error and the cause both have a value for the
	// same key, the top errors value will always take precedence, even
	// if the cause was attached to the error after the value was.

	err = New("boom", WithValue("color", "red"))
	err = Wrap(err, WithCause(New("io error", WithValue("color", "blue"))))
	assert.Equal(t, "red", Value(err, "color"))
}

func TestValues(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Values(nil))

	// error with no values should still be nil
	assert.Nil(t, Values(errors.New("boom")))

	// create an error chain with a few values attached, and a non-werror
	// in the middle.
	err := New("boom", WithValue("raw", "bam"), WithCode(4))
	err = &UnwrapperError{err}
	err = Wrap(err, WithValue("color", "red"))

	values := Values(err)

	assert.Equal(t, map[interface{}]interface{}{
		"raw":      "bam",
		ErrKeyCode: codes.Code(4),
		"color":    "red",
	}, values)
}

func BenchmarkValues(b *testing.B) {
	// create an error chain with a few values attached
	// and a non-werror in the middle
	err := New("boom", WithValue("raw", "bam"), WithCode(4))
	err = &UnwrapperError{err}
	err = Wrap(err, WithValue("color", "red"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Values(err)
	}
}

func TestCode(t *testing.T) {
	// nil -> 200
	assert.Equal(t, codes.Code(200), Code(nil))

	// default to 500
	assert.Equal(t, codes.Code(500), Code(errors.New("boom")))

	// set with wrapper
	assert.Equal(t, codes.Code(404), Code(New("boom", WithCode(404))))

	// works when value is deep in stack
	err := New("bam", WithCode(404))
	err = &UnwrapperError{err}
	err = Wrap(err, WithValue("raw", "yikes"))
	assert.Equal(t, codes.Code(404), Code(err))
}

func TestCause(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Cause(nil))

	// no cause -> nil
	assert.Nil(t, Cause(New("boom")))

	// with cause
	root := errors.New("boom")
	err := New("yikes", WithCause(root))
	assert.EqualError(t, Cause(err), "boom")

	// with nil cause, should be no-op
	err = New("yikes", WithCause(nil))
	assert.Nil(t, Cause(err))
}
