// Package httpkit writes JSON responses, reads JSON requests, and turns an
// error into an HTTP answer with one body shape.
package httpkit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
)

// maxBodyBytes is the largest request body DecodeJSON reads.
const maxBodyBytes = 1 << 20

// Error is an error that a client can trip. Its Status and Code reach the
// client; a plain error answers 500 "internal".
type Error struct {
	// Status is the HTTP status code.
	Status int
	// Code is a stable snake_case string that the frontend matches on.
	Code string
	// Message is an English text for logs and developers.
	Message string
}

// NewError returns an *Error.
func NewError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// errorBody is the JSON body of every error answer.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes v as a JSON body with status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// NoContent writes a 204 answer.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteError answers with err. An *Error in the chain gives the status and
// the code. Any other error is logged and answers 500.
func WriteError(w http.ResponseWriter, req *http.Request, logger *slog.Logger, err error) {
	if herr, ok := errors.AsType[*Error](err); ok {
		JSON(w, herr.Status, errorBody{Code: herr.Code, Message: herr.Message})
		return
	}
	logger.ErrorContext(req.Context(), "http: internal error",
		slog.String("method", req.Method),
		slog.String("path", req.URL.Path),
		slog.Any("err", err))
	JSON(w, http.StatusInternalServerError, errorBody{Code: "internal", Message: "internal error"})
}

// DecodeJSON reads the request body into dst. It refuses a body that is not
// application/json, that is larger than 1 MiB, or that has an unknown field.
// A JSON-only body is part of the CSRF defense: a plain HTML form cannot send
// one.
func DecodeJSON(w http.ResponseWriter, req *http.Request, dst any) error {
	mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return NewError(http.StatusUnsupportedMediaType, "unsupported_media_type",
			"the body must be application/json")
	}

	dec := json.NewDecoder(http.MaxBytesReader(w, req.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return NewError(http.StatusBadRequest, "invalid_json", err.Error())
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return NewError(http.StatusBadRequest, "invalid_json", "the body holds more than one JSON value")
	}
	return nil
}
