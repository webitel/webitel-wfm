package werror_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

func TestWrappers(t *testing.T) {
	tests := []struct {
		name       string
		wrapper    werror.Wrapper
		assertions func(*testing.T, error)
	}{
		{
			name:    "AppendMessage",
			wrapper: werror.AppendMessage("boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "bang: boom")
			},
		},
		{
			name:    "AppendMessagef",
			wrapper: werror.AppendMessagef("%s %s", "big", "boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "bang: big boom")
			},
		},
		{
			name:    "PrependMessage",
			wrapper: werror.PrependMessage("boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "boom: bang")
			},
		},
		{
			name:    "PrependMessagef",
			wrapper: werror.PrependMessagef("%s %s", "big", "boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "big boom: bang")
			},
		},
		{
			name:    "WithValue",
			wrapper: werror.WithValue("color", "red"),
			assertions: func(t *testing.T, err error) {
				assert.Equal(t, "red", werror.Value(err, "color"))
			},
		},
		{
			name:    "WithCode",
			wrapper: werror.WithCode(56),
			assertions: func(t *testing.T, err error) {
				assert.Equal(t, codes.Code(56), werror.Code(err))
			},
		},
		{
			name:    "WithCause",
			wrapper: werror.WithCause(errors.New("crash")),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, werror.Cause(err), "crash")
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
	assert.Nil(t, werror.Set(nil, "color", "red"))

	err := werror.Set(errors.New("bang"), "color", "red")
	assert.Equal(t, "red", werror.Value(err, "color"))
}
