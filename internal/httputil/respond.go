package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	apperrors "github.com/DelgadoElias/billax/internal/errors"
	"github.com/DelgadoElias/billax/internal/middleware"
	"github.com/DelgadoElias/billax/internal/validation"
)

// errorBody is the standard error envelope
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code      string                  `json:"code"`
	Message   string                  `json:"message"`
	RequestID string                  `json:"request_id"`
	Fields    []validation.FieldError `json:"fields,omitempty"`
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

	detail := errorDetail{Code: code, Message: msg, RequestID: requestID}

	// Check for ValidationError and expose structured field errors
	var ve *validation.ValidationError
	if errors.As(err, &ve) {
		detail.Fields = ve.Fields
	}

	// Never expose raw internal errors to clients
	if status == http.StatusInternalServerError {
		detail.Message = "an internal error occurred"
		detail.Code = "internal_error"
		detail.Fields = nil // don't expose fields on 500s
	}

	RespondJSON(w, status, errorBody{
		Error: detail,
	})
}

func RespondCreated(w http.ResponseWriter, r *http.Request, v any) {
	RespondJSON(w, http.StatusCreated, v)
}

func RespondOK(w http.ResponseWriter, v any) {
	RespondJSON(w, http.StatusOK, v)
}
