package shutdown

import (
	"errors"
	"fmt"
	"strings"
)

// cleanShutdown is a sentinel error used by the shutdown logic to indicate
// a clean shutdown, via context.Cause.
var cleanShutdown = errors.New("clean shutdown")

type shutdownError struct {
	handlerName string
	err         error
}

func (e shutdownError) Error() string {
	return fmt.Sprintf("shutdown handler %q: %v", e.handlerName, e.err)
}

func (e shutdownError) Unwrap() error {
	return e.err
}

type shutdownErrors struct {
	errors []error
}

func (e shutdownErrors) Unwrap() []error {
	return e.errors
}

func (e shutdownErrors) Error() string {
	switch len(e.errors) {
	case 0:
		return "no shutdown errors"
	case 1:
		return e.errors[0].Error()
	default:
		var buf strings.Builder
		buf.WriteString("multiple shutdown errors: ")
		for i, err := range e.errors {
			if i > 0 {
				buf.WriteString("; ")
			}

			buf.WriteString(err.Error())
		}

		return buf.String()
	}
}
