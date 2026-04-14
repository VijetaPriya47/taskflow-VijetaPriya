package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"taskflow-backend/internal/domain"
)

type AppError struct {
	Status  int               `json:"-"`
	Message string            `json:"-"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "error"
}

type errorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, errorResponse{Error: msg})
}

func writeAppError(w http.ResponseWriter, e AppError) {
	if e.Status <= 0 {
		e.Status = http.StatusBadRequest
	}
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.Status)
		if msg == "" {
			msg = "error"
		}
	}
	writeJSON(w, e.Status, errorResponse{Error: msg, Fields: e.Fields})
}

func writeServerError(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	var ae AppError
	if errors.As(err, &ae) {
		writeAppError(w, ae)
		return
	}

	switch {
	case errors.Is(err, domain.ErrUnauthorized),
		errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrInvalidToken):
		writeAppError(w, AppError{Status: http.StatusUnauthorized, Message: "unauthorized"})
		return
	case errors.Is(err, domain.ErrForbidden),
		errors.Is(err, domain.ErrProjectOwnerOnly),
		errors.Is(err, domain.ErrTaskDeleteForbidden):
		writeAppError(w, AppError{Status: http.StatusForbidden, Message: "forbidden"})
		return
	case errors.Is(err, domain.ErrNotFound):
		writeAppError(w, AppError{Status: http.StatusNotFound, Message: "not found"})
		return
	case errors.Is(err, domain.ErrConflict):
		writeAppError(w, AppError{Status: http.StatusConflict, Message: "conflict"})
		return
	case errors.Is(err, domain.ErrValidation):
		writeAppError(w, AppError{Status: http.StatusBadRequest, Message: "validation failed"})
		return
	default:
		_ = r // keep signature stable; request_id already logged elsewhere
		writeServerError(w)
		return
	}
}

func writeValidationError(w http.ResponseWriter, fields map[string]string) {
	if len(fields) == 0 {
		fields = map[string]string{"_": "invalid request"}
	}
	writeAppError(w, AppError{
		Status:  http.StatusBadRequest,
		Message: "validation failed",
		Fields:  fields,
	})
}

func ServiceUnavailable(msg string) error {
	if msg == "" {
		msg = "service unavailable"
	}
	return AppError{Status: http.StatusServiceUnavailable, Message: msg}
}

func BadJSON() error {
	return AppError{Status: http.StatusBadRequest, Message: "failed to parse JSON data"}
}

func InvalidURL() error {
	return AppError{Status: http.StatusBadRequest, Message: "invalid url"}
}

func Validation(fields map[string]string) error {
	return AppError{Status: http.StatusBadRequest, Message: "validation failed", Fields: fields}
}

func RequireUser(ctx context.Context) (string, error) {
	userID, _, ok := userFromContext(ctx)
	if !ok {
		return "", domain.ErrUnauthorized
	}
	return userID, nil
}
