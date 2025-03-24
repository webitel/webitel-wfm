package options

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

var (
	_ FieldsOption   = (*Read)(nil)
	_ DerivedOptions = (*Search)(nil)
)

type Read struct {
	fields
	derived

	user *model.SignedInUser
	id   int64
}

func NewRead(ctx context.Context, options ...Option) (*Read, error) {
	s := grpccontext.FromContext(ctx)
	if s.SignedInUser == nil {
		return nil, werror.Unauthenticated("can not find signed in user", werror.WithID("model.options.user"))
	}

	read := &Read{
		fields:  make([]string, 0),
		derived: make(map[string]*Derived),
		user:    s.SignedInUser,
	}

	for _, option := range options {
		if err := option(read); err != nil {
			return nil, err
		}
	}

	return read, nil
}

func (r *Read) User() *model.SignedInUser {
	return r.user
}

func (r *Read) ID() int64 {
	return r.id
}

func (r *Read) WithId(id int64) {
	r.id = id
}
