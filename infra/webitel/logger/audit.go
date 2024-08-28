package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/webitel/webitel-wfm/infra/pubsub"
	"github.com/webitel/webitel-wfm/internal/model"
)

var (
	rkFormat = "logger.%d.%s"
	exchange = pubsub.Exchange{
		Name:    "logger",
		Type:    pubsub.ExchangeTypeTopic,
		Durable: false,
	}
)

type Audit struct {
	svc *ConfigService
	pub *pubsub.Manager
}

func NewAudit(svc *ConfigService, pub *pubsub.Manager) *Audit {
	return &Audit{
		svc: svc,
		pub: pub,
	}
}

func (a *Audit) Create(ctx context.Context, user *model.SignedInUser, records map[int64]any) error {
	return a.audit(ctx, ActionCreate, user, records)
}

func (a *Audit) Update(ctx context.Context, user *model.SignedInUser, records map[int64]any) error {
	return a.audit(ctx, ActionUpdate, user, records)
}

func (a *Audit) Delete(ctx context.Context, user *model.SignedInUser, records map[int64]any) error {
	return a.audit(ctx, ActionDelete, user, records)
}

func (a *Audit) audit(ctx context.Context, action Action, user *model.SignedInUser, records map[int64]any) error {
	ok, err := a.svc.Active(ctx, user.DomainId, user.Object)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	items := make([]Record, 0, len(records))
	for id, rec := range records {
		body, err := json.Marshal(rec)
		if err != nil {
			return err
		}

		item := Record{
			Id:       id,
			NewState: body,
		}

		items = append(items, item)
	}

	msg := Message{
		Records: items,
		RequiredFields: RequiredFields{
			UserId:     user.Id,
			UserIp:     "",
			DomainId:   user.DomainId,
			Action:     action,
			Date:       time.Now().Unix(),
			ObjectName: user.Object,
		},
	}

	if err = a.pub.Channel().Publish(ctx, exchange.Name, fmt.Sprintf(rkFormat, msg.DomainId, user.Object), msg.ToJson()); err != nil {
		return err
	}

	return nil
}
