package werror

import (
	"fmt"
	"sync"
)

type DBNoRowsError struct {
	id string
}

func NewDBNoRowsErr(id string) DBNoRowsError {
	return DBNoRowsError{
		id: id,
	}
}

func (e DBNoRowsError) Id() string {
	return e.id
}

func (e DBNoRowsError) RPCError() string {
	return fmt.Sprintf("entity does not exists or you do not have enough permissions to perform the operation")
}

func (e DBNoRowsError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}

type DBUniqueViolationError struct {
	id     string
	Column string
	Value  string
}

func NewDBUniqueViolationError(id, column, value string) DBUniqueViolationError {
	return DBUniqueViolationError{
		id:     id,
		Column: column,
		Value:  value,
	}
}

func (e DBUniqueViolationError) Id() string {
	return e.id
}

func (e DBUniqueViolationError) RPCError() string {
	return fmt.Sprintf("invalid input: entity [%s = %s] already exists", e.Column, e.Value)
}

func (e DBUniqueViolationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}

type DBForeignKeyViolationError struct {
	id              string
	Column          string
	Value           string
	ForeignKeyTable string
}

func NewDBForeignKeyViolationError(id, column, value, foreignKey string) DBForeignKeyViolationError {
	return DBForeignKeyViolationError{
		id:              id,
		Column:          column,
		Value:           value,
		ForeignKeyTable: foreignKey,
	}
}

func (e DBForeignKeyViolationError) Id() string {
	return e.id
}

func (e DBForeignKeyViolationError) RPCError() string {
	if e.ForeignKeyTable != "" {
		return fmt.Sprintf("invalid input: violates foreign key constraint: %s (%s) is still referenced by the parent table", e.Column, e.Value)
	}

	return fmt.Sprintf("invalid input: violates foreign key constraint: the %s (%s) isn't present in the parent table", e.Column, e.Value)

}

func (e DBForeignKeyViolationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}

var checkViolationErrorRegistry = map[string]string{}
var constraintMu sync.RWMutex

// RegisterConstraint register custom database check constraint (like "CHECK
// balance > 0").
// Postgres doesn't define a very useful message for constraint
// failures (new row for relation "accounts" violates check constraint), so you
// can define your own.
//   - name - should be the name of the constraint in the database.
//   - message - your own custom error message
//
// Panics if you attempt to register two constraints with the same name.
func RegisterConstraint(name, message string) {
	constraintMu.Lock()
	defer constraintMu.Unlock()
	if _, dup := checkViolationErrorRegistry[name]; dup {
		panic("RegisterConstraint called twice for name " + name)
	}

	checkViolationErrorRegistry[name] = message
}

type DBCheckViolationError struct {
	id    string
	Check string
}

func NewDBCheckViolationError(id, check string) DBCheckViolationError {
	return DBCheckViolationError{
		id:    id,
		Check: check,
	}
}

func (e DBCheckViolationError) Id() string {
	return e.id
}

func (e DBCheckViolationError) RPCError() string {
	return fmt.Sprintf("invalid input: violates check constraint [%s]: %s", e.Check, checkViolationErrorRegistry[e.Check])

}

func (e DBCheckViolationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}

type DBNotNullViolationError struct {
	id     string
	Column string
}

func NewDBNotNullViolationError(id, column string) DBNotNullViolationError {
	return DBNotNullViolationError{
		id:     id,
		Column: column,
	}
}

func (e DBNotNullViolationError) Id() string {
	return e.id
}

func (e DBNotNullViolationError) RPCError() string {
	return fmt.Sprintf("invalid input: violates not null constraint: column [%s] can not be null", e.Column)
}

func (e DBNotNullViolationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}

type DBEntityConflictError struct {
	id string
}

func NewDBEntityConflictError(id string) DBEntityConflictError {
	return DBEntityConflictError{
		id: id,
	}
}

func (e DBEntityConflictError) Id() string {
	return e.id
}

func (e DBEntityConflictError) RPCError() string {
	return fmt.Sprintf("found more then one requested entity")
}

func (e DBEntityConflictError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}

type DBInternalError struct {
	id     string
	Reason error
}

func NewDBInternalError(id string, reason error) DBInternalError {
	return DBInternalError{
		id:     id,
		Reason: reason,
	}
}

func (e DBInternalError) Id() string {
	return e.id
}

func (e DBInternalError) RPCError() string {
	return fmt.Sprintf("internal server error: %s", e.Reason)
}

func (e DBInternalError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.Reason.Error())
}
