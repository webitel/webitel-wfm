package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/storage"
)

// TODO: add validation for cause.id
// for i, v := range in {
//		if err := p.store.ReadPauseCause(ctx, v.DomainId, v.Cause.Token); err != nil {
//			return err.SetDetailedError(fmt.Sprintf("items[%d].cause.id: not found: %s", i, err.appError()))
//		}
// }

type PauseTemplateManager interface {
	CreatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) (int64, error)
	ReadPauseTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.PauseTemplate, error)
	SearchPauseTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.PauseTemplate, bool, error)
	UpdatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) error
	DeletePauseTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}
type PauseTemplate struct {
	storage storage.PauseTemplateManager
}

func NewPauseTemplate(storage storage.PauseTemplateManager) *PauseTemplate {
	return &PauseTemplate{
		storage: storage,
	}
}

func (p *PauseTemplate) CreatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) (int64, error) {
	id, err := p.storage.CreatePauseTemplate(ctx, user, in)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (p *PauseTemplate) ReadPauseTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.PauseTemplate, error) {
	out, err := p.storage.ReadPauseTemplate(ctx, user, id, fields)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (p *PauseTemplate) SearchPauseTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.PauseTemplate, bool, error) {
	out, err := p.storage.SearchPauseTemplate(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(search.Limit(), out)

	return out, next, nil
}

func (p *PauseTemplate) UpdatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) error {
	if err := p.storage.UpdatePauseTemplate(ctx, user, in); err != nil {
		return err
	}

	return nil
}

func (p *PauseTemplate) DeletePauseTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	out, err := p.storage.DeletePauseTemplate(ctx, user, id)
	if err != nil {
		return 0, err
	}

	return out, nil
}
