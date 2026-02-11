package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kareempaes/planning/internal/model"
)

// ErrorBody is the standard error response envelope.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail carries a machine-readable code and human-readable message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	var ve *model.ValidationError

	switch {
	case errors.As(err, &ve):
		writeJSON(w, http.StatusUnprocessableEntity, ErrorBody{
			Error: ErrorDetail{Code: "validation_error", Message: ve.Error()},
		})
	case errors.Is(err, model.ErrNotFound):
		writeJSON(w, http.StatusNotFound, ErrorBody{
			Error: ErrorDetail{Code: "not_found", Message: "resource not found"},
		})
	case errors.Is(err, model.ErrConflict):
		writeJSON(w, http.StatusConflict, ErrorBody{
			Error: ErrorDetail{Code: "conflict", Message: "resource already exists"},
		})
	default:
		writeJSON(w, http.StatusInternalServerError, ErrorBody{
			Error: ErrorDetail{Code: "internal_error", Message: "internal server error"},
		})
	}
}

func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
