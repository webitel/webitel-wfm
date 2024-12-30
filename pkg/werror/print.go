package werror

import (
	"fmt"
	"io"
	"strings"
)

// Details returns err.Error() and any additional details.
// The details of each error cause chain will also be printed.
func Details(err error) string {
	if err == nil {
		return ""
	}

	var det string
	values := Values(err)
	for key, value := range values {
		if k, ok := key.(string); ok {
			det = fmt.Sprintf("%s; %s = %s", det, k, value)
		}
	}

	msg := err.Error() + det
	if id := ID(err); id != "" {
		msg = id + ": " + msg
	}

	if cause := Cause(err); cause != nil {
		msg += "; caused by: " + Details(cause)
	}

	return msg
}

// Format adapts errors to fmt.Formatter interface.
// It's intended to be used help error impls implement fmt.Formatter, e.g.:
//
//	    func (e *myErr) Format(f fmt.State, verb rune) {
//		     Format(f, verb, e)
//	    }
func Format(s fmt.State, verb rune, err error) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, Details(err))

			return
		}

		fallthrough
	case 's':
		io.WriteString(s, msgWithCauses(err))
	case 'q':
		fmt.Fprintf(s, "%q", err.Error())
	}
}

func msgWithCauses(err error) string {
	messages := make([]string, 0, 5)
	for err != nil {
		if ce := err.Error(); ce != "" {
			messages = append(messages, ce)
		}

		err = Cause(err)
	}

	return strings.Join(messages, ": ")
}
