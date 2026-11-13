package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

type Holiday struct {
	Date pgtype.Date
	Name string
}

func (h *Holiday) MarshalProto() *pb.Holiday {
	return &pb.Holiday{
		Date: DateMillis(h.Date),
		Name: h.Name,
	}
}

// MarshalProto keys the week by day as the Add shifts template does, every day present.
func (c *ScheduleCalendar) MarshalProto() *pb.ScheduleCalendar {
	if c == nil {
		return nil
	}

	week := make(map[int64]*pb.CalendarDay, len(c.Week))
	for _, day := range c.Week {
		ranges := make([]*pb.CalendarRange, 0, len(day.Ranges))
		for _, r := range day.Ranges {
			ranges = append(ranges, &pb.CalendarRange{Start: int32(r.Start), End: int32(r.End)})
		}

		week[int64(day.Day)] = &pb.CalendarDay{Ranges: ranges}
	}

	return &pb.ScheduleCalendar{TimeZone: c.TimeZone, Week: week}
}

// ScheduleCalendar is the calendar of a schedule: the IANA zone its times are shown in,
// and its working week from Sunday to Saturday.
type ScheduleCalendar struct {
	TimeZone string
	Week     [7]CalendarWeekDay
}

// CalendarWeekDay holds the hours the calendar works on a day of week, in minutes from
// midnight; a day without any is off.
type CalendarWeekDay struct {
	Day    time.Weekday
	Ranges []CalendarRange
}

type CalendarRange struct {
	Start int16
	End   int16
}
