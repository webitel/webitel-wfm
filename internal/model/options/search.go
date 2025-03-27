package options

import (
	"context"
	"database/sql/driver"
	"slices"
	"strings"

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

type Substring string

func (s Substring) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "", nil
	}

	// TODO: escape(%)
	v := string(s)
	const escape = "\\" // https://postgrespro.ru/docs/postgresql/12/functions-matching#FUNCTIONS-LIKE

	// escape control '_' (single char entry)
	v = strings.ReplaceAll(v, "_", escape+"_")

	// propagate '?' char for PostgreSQL purpose
	v = strings.ReplaceAll(v, "?", "_")

	// escape control '%' (any char(s) or none)
	v = strings.ReplaceAll(v, "%", escape+"%")

	return v, nil
}

type Search struct {
	fields
	orderBy
	derived

	user *model.SignedInUser

	ids   []int64
	query Substring
	page  int
	size  int

	// TODO: parse CEL expressions
	filter map[string]any
}

func NewSearch(ctx context.Context, options ...Option) (*Search, error) {
	s := grpccontext.FromContext(ctx)
	if s.SignedInUser == nil {
		return nil, werror.Unauthenticated("can not find signed in user", werror.WithID("model.options.user"))
	}

	search := &Search{
		fields:  make(fields, 0),
		orderBy: make(orderBy),
		derived: make(derived),
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

func (s *Search) Query() Substring {
	return s.query
}

func (s *Search) Size() int {
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

func (s *Search) Page() int {
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

func (s *Search) Offset() int {
	return s.Size() * (s.Page() - 1)
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
	s.query = Substring(term)
}

func (s *Search) WithPagination(size int32, page int32) {
	s.size = int(size)
	s.page = int(page)
}

func (s *Search) PopulateFromRead(r *Read) *Search {
	s.fields = r.fields
	s.derived = r.derived

	return s
}
