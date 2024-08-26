package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

// TODO: add signed user id to cache key
// TODO: add validation for cause.id
// for i, v := range in {
//		if err := p.store.ReadPauseCause(ctx, v.DomainId, v.Cause.Token); err != nil {
//			return err.SetDetailedError(fmt.Sprintf("items[%d].cause.id: not found: %s", i, err.appError()))
//		}
// }
// TODO: add validation for pause_template_id

type PauseTemplateManager interface {
	CreatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) (int64, error)
	SearchPauseTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.PauseTemplate, error)
	UpdatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) error
	DeletePauseTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)

	SearchPauseTemplateCause(ctx context.Context, user *model.SignedInUser, pauseTemplateId int64, search *model.SearchItem) ([]*model.PauseTemplateCause, error)
	UpdatePauseTemplateCauseBulk(ctx context.Context, user *model.SignedInUser, pauseTemplateId int64, in []*model.PauseTemplateCause) error
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

func (p *PauseTemplate) ReadPauseTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.PauseTemplate, error) {
	items, err := p.svc.SearchPauseTemplate(ctx, user, search)
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, werror.NewDBEntityConflictError("service.pause_template.read.conflict")
	}

	if len(items) == 0 {
		return nil, werror.NewDBNoRowsErr("service.pause_template.read")
	}

	return items[0], nil
}

func (p *PauseTemplate) SearchPauseTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.PauseTemplate, bool, error) {
	items, err := p.svc.SearchPauseTemplate(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	var next bool
	if len(items) == int(search.Limit()) {
		next = true
		items = items[:search.Limit()-1]
	}

	return items, next, nil
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

func (p *PauseTemplate) SearchPauseTemplateCause(ctx context.Context, user *model.SignedInUser, pauseTemplateId int64, search *model.SearchItem) ([]*model.PauseTemplateCause, bool, error) {
	items, err := p.svc.SearchPauseTemplateCause(ctx, user, pauseTemplateId, search)
	if err != nil {
		return nil, false, err
	}

	var next bool
	if len(items) == int(search.Limit()) {
		next = true
	}

	return items, next, nil
}

func (p *PauseTemplate) UpdatePauseTemplateCauseBulk(ctx context.Context, user *model.SignedInUser, pauseTemplateId int64, in []*model.PauseTemplateCause) error {
	if err := p.svc.UpdatePauseTemplateCauseBulk(ctx, user, pauseTemplateId, in); err != nil {
		return err
	}

	return nil
}
