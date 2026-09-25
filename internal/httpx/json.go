package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxBodyBytes = 1 << 20

func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var payload T

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		var zero T
		return zero, decodeError(err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var zero T
		return zero, NewError(http.StatusBadRequest, "body must contain a single JSON object", err)
	}

	return payload, nil
}

func WriteJSON(w http.ResponseWriter, status int, payload any) error {
	return writeBody(w, status, "application/json", payload)
}

func writeBody(w http.ResponseWriter, status int, contentType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(body)

	return nil
}

func decodeError(err error) *Error {
	var maxBytesErr *http.MaxBytesError
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError

	switch {
	case errors.As(err, &maxBytesErr):
		return NewError(http.StatusRequestEntityTooLarge, fmt.Sprintf("body must not exceed %d bytes", maxBytesErr.Limit), err)
	case errors.Is(err, io.EOF):
		return NewError(http.StatusBadRequest, "body must not be empty", err)
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		return NewError(http.StatusBadRequest, "body contains malformed JSON", err)
	case errors.As(err, &typeErr):
		return NewError(http.StatusBadRequest, fmt.Sprintf("field %q has an invalid type", typeErr.Field), err)
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		return NewError(http.StatusBadRequest, "body contains "+strings.TrimPrefix(err.Error(), "json: "), err)
	default:
		return NewError(http.StatusBadRequest, "body could not be decoded", err)
	}
}
