package options

import (
	"context"
	"slices"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const DefaultSearchSize = 16

var (
	_ FieldsOption   = (*Search)(nil)
	_ OrderByOption  = (*Search)(nil)
	_ DerivedOptions = (*Search)(nil)
)

type Search struct {
	fields
	orderBy
	derived

	user *model.SignedInUser

	ids   []int64
	query string
	page  int64
	size  int64

	// TODO: parse CEL expressions
	filter map[string]any
}

func NewSearch(ctx context.Context, options ...Option) (*Search, error) {
	s := grpccontext.FromContext(ctx)
	if s.SignedInUser == nil {
		return nil, werror.Unauthenticated("can not find signed in user", werror.WithID("model.options.user"))
	}

	search := &Search{
		fields:  make([]string, 0),
		orderBy: make(map[string]OrderDirection),
		derived: make(map[string]*Derived),
		user:    s.SignedInUser,
		ids:     make([]int64, 0),
	}

	for _, option := range options {
		if err := option(search); err != nil {
			return nil, err
		}
	}

	return search, nil
}

func (s *Search) User() *model.SignedInUser {
	return s.user
}

func (s *Search) IDs() []int64 {
	return s.ids
}

func (s *Search) Size() int64 {
	if s == nil {
		return DefaultSearchSize
	}

	switch {
	case s.size < 0:
		return -1
	case s.size > 0:
		// TODO: check for too big values
		return s.size
	case s.size == 0:
		return DefaultSearchSize
	}

	return s.size
}

func (s *Search) Page() int64 {
	if s != nil {
		if s.Size() > 0 {
			if s.page > 0 {
				return s.page
			}

			return 1
		}
	}

	return 0
}

func (s *Search) WithId(id int64) {
	if !slices.Contains(s.ids, id) {
		s.ids = append(s.ids, id)
	}
}

func (s *Search) WithIds(ids []int64) {
	for _, id := range ids {
		s.WithId(id)
	}
}

func (s *Search) WithSearch(term string) {
	s.query = term
}

func (s *Search) WithPagination(size int32, page int32) {
	s.size = int64(size)
	s.page = int64(page)
}
