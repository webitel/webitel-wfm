package options

import (
	"strings"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

var ErrInsufficientRequestCapabilities = werror.InvalidArgument("insufficient request option capabilities", werror.WithID("model.options"))

type Option func(options any) error

func WithID(id int64) Option {
	return func(options any) error {
		v, ok := options.(interface{ WithId(int64) })
		if !ok {
			if err := WithIDs(id)(options); err != nil {
				return werror.Wrap(ErrInsufficientRequestCapabilities, werror.WithCause(err), werror.WithValue("option", "id"))
			}
		}

		v.WithId(id)

		return nil
	}
}

func WithIDs(id ...int64) Option {
	return func(options any) error {
		v, ok := options.(interface{ WithIds([]int64) })
		if !ok {
			return werror.Wrap(ErrInsufficientRequestCapabilities, werror.WithValue("option", "ids"))
		}

		v.WithIds(id)

		return nil
	}
}

func WithFields(fields []string) Option {
	return func(options any) error {
		for _, f := range fields {
			fieldParts := strings.Split(f, ".")
			if err := processField(options, fieldParts); err != nil {
				return err
			}
		}

		return nil
	}
}

func WithSearch(term string) Option {
	return func(options any) error {
		if term != "" {
			v, ok := options.(interface{ WithSearch(string) })
			if !ok {
				return werror.Wrap(ErrInsufficientRequestCapabilities, werror.WithValue("option", "search"))
			}

			v.WithSearch(term)
		}

		return nil
	}
}

func WithOrder(field ...string) Option {
	return func(options any) error {
		for _, f := range field {
			fieldName, fieldOrder := order(f)
			fieldParts := strings.Split(fieldName, ".")
			if err := processOrderByField(options, fieldParts, fieldOrder); err != nil {
				return err
			}
		}

		return nil
	}
}

func WithPagination(page, size int32) Option {
	return func(options any) error {
		v, ok := options.(interface{ WithPagination(int32, int32) })
		if !ok {
			return werror.Wrap(ErrInsufficientRequestCapabilities, werror.WithValue("option", "pagination"))
		}

		if page == 0 {
			page = 1
		}

		if size < 0 {
			size = -1
		}

		v.WithPagination(page, size)

		return nil
	}
}

// processField recursively processes fields and derived structures
func processField(options any, fieldParts []string) error {
	v, ok := options.(FieldsOption)
	if !ok {
		return werror.Wrap(ErrInsufficientRequestCapabilities, werror.WithValue("option", "fields"))
	}

	firstPart := fieldParts[0]
	v.WithField(firstPart)

	remainingParts := fieldParts[1:]
	if len(remainingParts) > 0 {
		// Check if we need to create or traverse into a nested derived struct
		d, ok := options.(DerivedOptions)
		if !ok {
			return werror.Wrap(ErrInsufficientRequestCapabilities, werror.WithValue("option", "derived_fields"))
		}

		derived := d.DerivedByName(firstPart)
		if derived == nil {
			derived = &Derived{}
		}

		// Now handle the next level (remainingParts) in the derived struct
		// Recursively call processField for the next part of the nested field
		if err := processField(derived, fieldParts[1:]); err != nil {
			return err
		}

		d.WithDerived(firstPart, derived)
	}

	return nil
}

// processOrderByField is a recursive function that processes each part of the field
func processOrderByField(options any, fieldParts []string, direction builder.OrderDirection) error {
	v, ok := options.(OrderByOption)
	if !ok {
		return werror.Wrap(ErrInsufficientRequestCapabilities, werror.WithValue("option", "order_by"))
	}

	// Recursive case: It's a nested field
	// Look for the first part of the nested field
	firstPart := fieldParts[0]
	remainingParts := fieldParts[1:]
	// If this was the last part, assign the order direction at this level
	if len(remainingParts) == 0 {
		v.WithOrderBy(firstPart, direction)

		return nil
	}

	d, ok := options.(DerivedOptions)
	if !ok {
		return werror.Wrap(ErrInsufficientRequestCapabilities, werror.WithValue("option", "derived_order_by"))
	}

	derived := d.DerivedByName(firstPart)
	if derived == nil {
		derived = &Derived{}
	}

	// Now handle the next level (remainingParts) in the derived struct
	// Recursively call processField for the next part of the nested field
	if err := processOrderByField(derived, remainingParts, direction); err != nil {
		return err
	}

	d.WithDerived(firstPart, derived)

	return nil
}

func order(s string) (field string, sort builder.OrderDirection) {
	if s[0] == '+' || s[0] == 32 {
		return s[1:], builder.OrderDirectionASC
	}

	if s[0] == '-' {
		return s[1:], builder.OrderDirectionDESC
	}

	return s, builder.OrderDirectionASC
}
