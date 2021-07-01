// The errors package provides additional error primitives.
package errors

import (
	"errors"
	"strings"
)

func New(text string) error {
	return errors.New(text)
}

func Unwrap(err error) error {
	return errors.Unwrap(err)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Errors is a list of errors.
type Errors []error

// Errors formats the list by separating each message with a newline. Each
// produced line, including lines within messages, is prefixed with a tab.
func (errs Errors) Error() string {
	switch len(errs) {
	case 0:
		return "no errors"
	case 1:
		return errs[0].Error()
	default:
		var buf strings.Builder
		buf.WriteString("multiple errors:")
		for _, err := range errs {
			buf.WriteString("\n\t")
			msg := err.Error()
			msg = strings.ReplaceAll(msg, "\n", "\n\t")
			buf.WriteString(msg)
		}
		return buf.String()
	}
}

// Append returns errs with each err appended to it. Arguments that are nil are
// skipped.
func (errs Errors) Append(err ...error) Errors {
	for _, err := range err {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// Return prepares errs to be returned by a function by returning nil if errs is
// empty.
func (errs Errors) Return() error {
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// Union receives a number of errors and combines them into one Errors. Any errs
// that are Errors are concatenated directly. Returns nil if all errs are nil or
// empty.
func Union(errs ...error) error {
	var e Errors
	for _, err := range errs {
		switch err := err.(type) {
		case nil:
			continue
		case Errors:
			for _, err := range err {
				if err != nil {
					e = append(e, err)
				}
			}
		default:
			e = append(e, err)
		}
	}
	return e.Return()
}
