package httpapi

import (
	"net/http"
	"strconv"
	"timber-release-gate/internal/application"
	"timber-release-gate/internal/domain"
)

func (a *API) timeline(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		method(w)
		return
	}
	v, e := a.s.Timeline(id)
	if e != nil {
		errWrite(w, e)
		return
	}
	write(w, 200, struct {
		domain.Timeline
		AuditVerified bool `json:"auditVerified"`
	}{v, true})
}
func (a *API) risk(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		method(w)
		return
	}
	filter := application.RiskFilter{ComponentCode: r.URL.Query().Get("componentCode"), Severity: domain.Severity(r.URL.Query().Get("severity")), Status: domain.FindingStatus(r.URL.Query().Get("status"))}
	if raw, ok := r.URL.Query()["covered"]; ok {
		if len(raw) != 1 {
			errWrite(w, &domain.Error{Code: "INVALID_FILTER", Message: "covered筛选值无效"})
			return
		}
		value, err := strconv.ParseBool(raw[0])
		if err != nil {
			errWrite(w, &domain.Error{Code: "INVALID_FILTER", Message: "covered筛选值只允许true或false"})
			return
		}
		filter.Covered = &value
	}
	v, e := a.s.QueryRisk(id, filter)
	if e != nil {
		errWrite(w, e)
		return
	}
	write(w, 200, v)
}
func (a *API) certificate(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		method(w)
		return
	}
	v, e := a.s.Certificate(id)
	if e != nil {
		errWrite(w, e)
		return
	}
	write(w, 200, struct {
		*domain.WorkReleaseCertificate
		AuditVerified    bool `json:"auditVerified"`
		ManifestVerified bool `json:"manifestVerified"`
	}{v, true, true})
}
