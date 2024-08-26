package werror

import "fmt"

type ValidationError struct {
	id         string
	Type       string
	Field      string
	Constraint string
	Message    string
}

func NewValidationError(id, messageType, field, constraint, message string) ValidationError {
	return ValidationError{
		id:         id,
		Type:       messageType,
		Field:      field,
		Constraint: constraint,
		Message:    message,
	}
}

func (e ValidationError) Id() string {
	return e.id
}

func (e ValidationError) RPCError() string {
	return fmt.Sprintf("validate message [%s]: %s[%s]: %s", e.Type, e.Field, e.Constraint, e.Message)
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Id(), e.RPCError())
}
