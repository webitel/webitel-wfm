package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
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
	SearchPauseTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.PauseTemplate, error)
	UpdatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) error
	DeletePauseTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}

type PauseTemplate struct {
	svc PauseTemplateManager
}

func NewPauseTemplate(svc PauseTemplateManager) *PauseTemplate {
	return &PauseTemplate{
		svc: svc,
	}
}

func (p *PauseTemplate) CreatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) (int64, error) {
	id, err := p.svc.CreatePauseTemplate(ctx, user, in)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (p *PauseTemplate) ReadPauseTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.PauseTemplate, error) {
	out, err := p.svc.ReadPauseTemplate(ctx, user, id, fields)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (p *PauseTemplate) SearchPauseTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.PauseTemplate, bool, error) {
	out, err := p.svc.SearchPauseTemplate(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(search.Limit(), out)

	return out, next, nil
}

func (p *PauseTemplate) UpdatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) error {
	if err := p.svc.UpdatePauseTemplate(ctx, user, in); err != nil {
		return err
	}

	return nil
}

func (p *PauseTemplate) DeletePauseTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	out, err := p.svc.DeletePauseTemplate(ctx, user, id)
	if err != nil {
		return 0, err
	}

	return out, nil
}
