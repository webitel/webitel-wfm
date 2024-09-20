package model

import pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"

type WorkingCondition struct {
	DomainRecord

	Name             string      `json:"name" db:"name"`
	Description      *string     `json:"description" db:"description"`
	WorkdayHours     *int32      `json:"workday_hours" db:"workday_hours"`
	WorkdaysPerMonth *int32      `json:"workdays_per_month" db:"workdays_per_month"`
	Vacation         *int32      `json:"vacation" db:"vacation"`
	SickLeaves       *int32      `json:"sick_leaves" db:"sick_leaves"`
	DaysOff          *int32      `json:"days_off" db:"days_off"`
	PauseDuration    *int32      `json:"pause_duration" db:"pause_duration"`
	PauseTemplate    LookupItem  `json:"pause_template" db:"pause_template,json"`
	ShiftTemplate    *LookupItem `json:"shift_template" db:"shift_template,json"`
}

func (w *WorkingCondition) MarshalProto() *pb.WorkingCondition {
	out := &pb.WorkingCondition{
		Id:               w.Id,
		DomainId:         w.DomainId,
		CreatedBy:        w.CreatedBy.MarshalProto(),
		UpdatedBy:        w.UpdatedBy.MarshalProto(),
		Name:             w.Name,
		Description:      w.Description,
		WorkdayHours:     w.WorkdayHours,
		WorkdaysPerMonth: w.WorkdaysPerMonth,
		Vacation:         w.Vacation,
		SickLeaves:       w.SickLeaves,
		DaysOff:          w.DaysOff,
		PauseDuration:    w.PauseDuration,
		PauseTemplate: &pb.LookupEntity{
			Id:   w.PauseTemplate.Id,
			Name: w.PauseTemplate.Name,
		},
	}

	if w.ShiftTemplate != nil {
		out.ShiftTemplate = &pb.LookupEntity{
			Id:   w.ShiftTemplate.Id,
			Name: w.ShiftTemplate.Name,
		}
	}

	if !w.CreatedAt.Time.IsZero() {
		out.CreatedAt = w.CreatedAt.Time.UnixMilli()
	}

	if !w.UpdatedAt.Time.IsZero() {
		out.UpdatedAt = w.UpdatedAt.Time.UnixMilli()
	}

	return out
}
