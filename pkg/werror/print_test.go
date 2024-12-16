package werror

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetails(t *testing.T) {
	// nil -> empty
	assert.Empty(t, Details(nil))

	err := New("bang", WithValue("raw", "stay calm"))
	det := Details(err)
	assert.Equal(t, "bang", err.Error())
	assert.Contains(t, det, "raw = stay calm")
}
