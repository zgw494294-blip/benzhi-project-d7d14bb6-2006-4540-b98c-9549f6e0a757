package httpapi

import (
	"net/http"
	"timber-release-gate/internal/domain"
)

type Envelope struct {
	Data      any        `json:"data,omitempty"`
	Error     *ErrorBody `json:"error,omitempty"`
	RequestID string     `json:"requestID,omitempty"`
}
type ErrorBody struct {
	Code          string              `json:"code"`
	Message       string              `json:"message"`
	State         domain.DossierState `json:"state,omitempty"`
	Version       uint64              `json:"version,omitempty"`
	AuditVerified *bool               `json:"auditVerified,omitempty"`
}

func statusFor(code string) int {
	switch code {
	case "NOT_FOUND":
		return http.StatusNotFound
	case "VERSION_CONFLICT", "IDEMPOTENCY_CONFLICT":
		return http.StatusConflict
	case "AUDIT_INTEGRITY_ERROR", "MANIFEST_INTEGRITY_ERROR", "CERTIFICATE_INTEGRITY_ERROR":
		return http.StatusConflict
	case "INVALID_STATE", "FROZEN":
		return http.StatusUnprocessableEntity
	default:
		return http.StatusBadRequest
	}
}
func writeError(w http.ResponseWriter, e error) {
	x := errorBody(e)
	write(w, statusFor(x.Code), Envelope{Error: &x})
}
func errorBody(e error) ErrorBody {
	if x, ok := e.(*domain.Error); ok {
		body := ErrorBody{Code: x.Code, Message: x.Message, State: x.State, Version: x.Version}
		if x.Code == "AUDIT_INTEGRITY_ERROR" || x.Code == "MANIFEST_INTEGRITY_ERROR" || x.Code == "CERTIFICATE_INTEGRITY_ERROR" {
			verified := false
			body.AuditVerified = &verified
		}
		return body
	}
	return ErrorBody{Code: "BUSINESS_ERROR", Message: e.Error()}
}
func writeData(w http.ResponseWriter, data any) { write(w, http.StatusOK, Envelope{Data: data}) }
func noContent(w http.ResponseWriter)           { w.WriteHeader(http.StatusNoContent) }
func requestID(r *http.Request) string {
	if v := r.Header.Get("X-Request-ID"); v != "" {
		return v
	}
	return "generated"
}
