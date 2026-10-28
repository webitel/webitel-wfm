package options

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
)

// The after-write intent is off by default, is set explicitly, and travels
// from a Read to the Search built from it.
func TestAfterWrite(t *testing.T) {
	ctx := grpccontext.SetUser(context.Background(), &model.SignedInUser{Id: 1, DomainId: 1})

	read, err := NewRead(ctx, WithID(7))
	require.NoError(t, err)
	assert.False(t, read.AfterWrite(), "a new read claims to follow a write")

	search, err := NewSearch(ctx)
	require.NoError(t, err)
	assert.False(t, search.PopulateFromRead(read).AfterWrite(), "a search built from a plain read claims to follow a write")

	assert.True(t, read.WithAfterWrite().AfterWrite(), "WithAfterWrite did not mark the read")

	search, err = NewSearch(ctx)
	require.NoError(t, err)
	assert.True(t, search.PopulateFromRead(read).AfterWrite(), "PopulateFromRead dropped the after-write intent")

	search, err = NewSearch(ctx)
	require.NoError(t, err)
	search.WithAfterWrite()
	assert.True(t, search.AfterWrite(), "WithAfterWrite did not mark the search")
}
