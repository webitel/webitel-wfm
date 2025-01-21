package cluster

import (
	"strings"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
)

// NodeCheckErrors is a set of checked nodes errors.
// This type can be used in errors.As/Is as it implements errors.Unwrap method.
type NodeCheckErrors []NodeCheckError

func (n NodeCheckErrors) Error() string {
	var b strings.Builder
	for i, err := range n {
		if i > 0 {
			b.WriteByte('\n')
		}

		b.WriteString(err.Error())
	}

	return b.String()
}

// Unwrap is a helper for errors.Is/errors.As functions.
func (n NodeCheckErrors) Unwrap() []error {
	errs := make([]error, len(n))
	for i, err := range n {
		errs[i] = err
	}

	return errs
}

// NodeCheckError implements `error` and contains information about unsuccessful node check.
type NodeCheckError struct {
	node dbsql.Node
	err  error
}

// Node returns dead node instance.
func (n NodeCheckError) Node() dbsql.Node {
	return n.node
}

// Error implements `error` interface.
func (n NodeCheckError) Error() string {
	if n.err == nil {
		return ""
	}

	return n.err.Error()
}

// Unwrap returns underlying error.
func (n NodeCheckError) Unwrap() error {
	return n.err
}
