package werror

import "fmt"

type WorkingScheduleUpdateDraftError struct {
	id string
}

func NewWorkingScheduleUpdateDraftErr(id string) WorkingScheduleUpdateDraftError {
	return WorkingScheduleUpdateDraftError{
		id: id,
	}
}

func (e WorkingScheduleUpdateDraftError) Id() string {
	return e.id
}

func (e WorkingScheduleUpdateDraftError) RPCError() string {
	return fmt.Sprintf("stored procedure (%s) doesn't exist")
}

func (e WorkingScheduleUpdateDraftError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}
