// Package tomlerror adds actionable diagnostics to strict TOML decoder errors.
package tomlerror

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// unknownFieldsError preserves the original strict decoding error with readable field locations.
type unknownFieldsError struct {
	// cause is the original decoder error, retained for errors.As and errors.Is.
	cause error
	// fields identifies all unknown keys and their positions.
	fields []toml.DecodeError
}

// Error describes each unknown TOML field and its source location.
func (e *unknownFieldsError) Error() string {
	var message strings.Builder
	if len(e.fields) == 1 {
		message.WriteString("unknown TOML field:")
	} else {
		message.WriteString("unknown TOML fields:")
	}

	for _, field := range e.fields {
		line, column := field.Position()
		fmt.Fprintf(&message, "\n  line %d, column %d: %q", line, column, strings.Join(field.Key(), "."))
	}

	return message.String()
}

// Unwrap returns the original TOML error for inspection by callers.
func (e *unknownFieldsError) Unwrap() error {
	return e.cause
}

// Explain replaces a generic strict-mode message with the unknown keys and their positions.
func Explain(err error) error {
	var unknown *toml.StrictMissingError
	if !errors.As(err, &unknown) || len(unknown.Errors) == 0 {
		return err
	}

	return &unknownFieldsError{cause: err, fields: unknown.Errors}
}
