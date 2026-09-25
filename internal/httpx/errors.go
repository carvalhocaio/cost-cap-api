package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/carvalhocaio/cost-cap-api/internal/validation"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) error

type Adapter func(HandlerFunc) http.Handler

type Error struct {
	Status int
	Detail string
	Err    error
}

func NewError(status int, detail string, err error) *Error {
	return &Error{Status: status, Detail: detail, Err: err}
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Detail
	}

	return e.Detail + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func NewAdapter(logger *slog.Logger) Adapter {
	return func(handler HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := handler(w, r); err != nil {
				renderError(w, r, logger, err)
			}
		})
	}
}

func renderError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	var validationErrs validation.Errors
	var httpErr *Error

	switch {
	case errors.As(err, &validationErrs):
		writeProblem(w, r, Problem{
			Status: http.StatusUnprocessableEntity,
			Detail: "request validation failed",
			Errors: toFieldErrors(validationErrs),
		})
	case errors.As(err, &httpErr):
		if httpErr.Status >= http.StatusInternalServerError {
			logFailure(logger, r, err)
		}
		WriteProblem(w, r, httpErr.Status, httpErr.Detail)
	default:
		logFailure(logger, r, err)
		WriteProblem(w, r, http.StatusInternalServerError, "")
	}
}

func logFailure(logger *slog.Logger, r *http.Request, err error) {
	logger.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
}

func toFieldErrors(errs validation.Errors) []FieldError {
	fields := make([]FieldError, len(errs))
	for i, fieldErr := range errs {
		fields[i] = FieldError{Field: fieldErr.Field, Message: fieldErr.Message}
	}

	return fields
}
