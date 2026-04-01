package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	apperrors "github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/middleware"
)

// errorBody is the standard error envelope
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func RespondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.RequestIDFromContext(r.Context())
	status := apperrors.HTTPStatusFor(err)
	code := apperrors.CodeFor(err)

	// Unwrap DomainError for message override
	var de *apperrors.DomainError
	msg := err.Error()
	if errors.As(err, &de) {
		msg = de.Message
		code = de.Code
	}

	// Never expose raw internal errors to clients
	if status == http.StatusInternalServerError {
		msg = "an internal error occurred"
		code = "internal_error"
	}

	RespondJSON(w, status, errorBody{
		Error: errorDetail{Code: code, Message: msg, RequestID: requestID},
	})
}

func RespondCreated(w http.ResponseWriter, r *http.Request, v any) {
	RespondJSON(w, http.StatusCreated, v)
}

func RespondOK(w http.ResponseWriter, v any) {
	RespondJSON(w, http.StatusOK, v)
}
