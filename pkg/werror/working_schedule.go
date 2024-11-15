package werror

import "fmt"

type WorkingScheduleUpdateDraftError struct {
	id    string
	State string
}

func NewWorkingScheduleUpdateDraftErr(id, state string) WorkingScheduleUpdateDraftError {
	return WorkingScheduleUpdateDraftError{
		id:    id,
		State: state,
	}
}

func (e WorkingScheduleUpdateDraftError) Id() string {
	return e.id
}

func (e WorkingScheduleUpdateDraftError) RPCError() string {
	return fmt.Sprintf("working schedule can only be updated in a draft state; current state: %s", e.State)
}

func (e WorkingScheduleUpdateDraftError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}
