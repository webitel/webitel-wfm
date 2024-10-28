package werror

import "fmt"

type ForecastProcedureNotFoundError struct {
	id        string
	Procedure string
}

func NewForecastProcedureNotFoundErr(id, procedure string) ForecastProcedureNotFoundError {
	return ForecastProcedureNotFoundError{
		id:        id,
		Procedure: procedure,
	}
}

func (e ForecastProcedureNotFoundError) Id() string {
	return e.id
}

func (e ForecastProcedureNotFoundError) RPCError() string {
	return fmt.Sprintf("stored procedure (%s) doesn't exist", e.Procedure)
}

func (e ForecastProcedureNotFoundError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}

type ForecastProcedureResultError struct {
	id      string
	Columns int
}

func NewForecastProcedureResultErr(id string, columns int) ForecastProcedureResultError {
	return ForecastProcedureResultError{
		id:      id,
		Columns: columns,
	}
}

func (e ForecastProcedureResultError) Id() string {
	return e.id
}

func (e ForecastProcedureResultError) RPCError() string {
	return fmt.Sprintf("forecast procedure execution result expected 2 columns, got %d", e.Columns)
}

func (e ForecastProcedureResultError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}
