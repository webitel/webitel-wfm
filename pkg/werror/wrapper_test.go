package werror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestWrappers(t *testing.T) {
	tests := []struct {
		name       string
		wrapper    Wrapper
		assertions func(*testing.T, error)
	}{
		{
			name:    "AppendMessage",
			wrapper: AppendMessage("boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "bang: boom")
			},
		},
		{
			name:    "AppendMessagef",
			wrapper: AppendMessagef("%s %s", "big", "boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "bang: big boom")
			},
		},
		{
			name:    "PrependMessage",
			wrapper: PrependMessage("boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "boom: bang")
			},
		},
		{
			name:    "PrependMessagef",
			wrapper: PrependMessagef("%s %s", "big", "boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "big boom: bang")
			},
		},
		{
			name:    "WithValue",
			wrapper: WithValue("color", "red"),
			assertions: func(t *testing.T, err error) {
				assert.Equal(t, "red", Value(err, "color"))
			},
		},
		{
			name:    "WithCode",
			wrapper: WithCode(56),
			assertions: func(t *testing.T, err error) {
				assert.Equal(t, codes.Code(56), Code(err))
			},
		},
		{
			name:    "WithCause",
			wrapper: WithCause(errors.New("crash")),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, Cause(err), "crash")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// nil -> nil
			assert.Nil(t, test.wrapper.Wrap(nil))

			err := test.wrapper.Wrap(errors.New("bang"))
			test.assertions(t, err)
		})
	}
}

func TestSet(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Set(nil, "color", "red"))

	err := Set(errors.New("bang"), "color", "red")
	assert.Equal(t, "red", Value(err, "color"))
}
