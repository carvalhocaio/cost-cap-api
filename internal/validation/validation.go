package validation

import "strings"

type FieldError struct {
	Field   string
	Message string
}

type Errors []FieldError

func (e *Errors) Add(field, message string) {
	*e = append(*e, FieldError{Field: field, Message: message})
}

func (e Errors) Error() string {
	messages := make([]string, len(e))
	for i, fieldErr := range e {
		messages[i] = fieldErr.Field + ": " + fieldErr.Message
	}

	return strings.Join(messages, "; ")
}

func (e Errors) OrNil() error {
	if len(e) == 0 {
		return nil
	}

	return e
}
