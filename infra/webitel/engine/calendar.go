package engine

import (
	"context"

	gogrpc "buf.build/gen/go/webitel/engine/grpc/go/_gogrpc"
	pb "buf.build/gen/go/webitel/engine/protocolbuffers/go"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/webitel"
	"github.com/webitel/webitel-wfm/internal/model"
)

type CalendarService struct {
	log *wlog.Logger
	cli gogrpc.CalendarServiceClient
}

func NewCalendarServiceClient(log *wlog.Logger, conn *webitel.ConnectionManager[*webitel.Connection]) (*CalendarService, error) {
	cli, err := conn.Connection()
	if err != nil {
		return nil, err
	}

	return &CalendarService{log: log, cli: gogrpc.NewCalendarServiceClient(cli.Client())}, nil
}

func (c *CalendarService) Holidays(ctx context.Context, calendarId int64) ([]*model.Holiday, error) {
	calendar, err := c.cli.ReadCalendar(ctx, &pb.ReadCalendarRequest{Id: calendarId})
	if err != nil {
		return nil, webitel.ParseError(err)
	}

	var excepts []*model.Holiday
	for _, e := range calendar.Excepts {
		if !e.Disabled {
			excepts = append(excepts, &model.Holiday{
				Date: model.NewDate(e.Date),
				Name: e.Name,
			})
		}
	}

	return excepts, nil
}