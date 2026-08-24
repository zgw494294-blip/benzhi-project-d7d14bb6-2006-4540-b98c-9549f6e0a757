package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"timber-release-gate/internal/application"
	"timber-release-gate/internal/domain"
	"time"
)

func (a *API) components(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var raw json.RawMessage
	if e := decode(r, &raw); e != nil {
		errWrite(w, e)
		return
	}
	inputs, bodyVersion, bodyActor, e := decodeComponents(raw)
	if e != nil {
		errWrite(w, e)
		return
	}
	d, e := a.s.AddComponents(id, inputs, requestVersion(r, bodyVersion), requestActor(r, bodyActor), key(r))
	if e != nil {
		errWrite(w, e)
		return
	}
	write(w, 200, d)
}

func decodeComponents(raw json.RawMessage) ([]application.ComponentInput, uint64, string, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, 0, "", &domain.Error{Code: "INVALID_INPUT", Message: "构件请求不能为空"}
	}
	if len(bytes.TrimSpace(raw)) > 0 && bytes.TrimSpace(raw)[0] == '[' {
		var list []application.ComponentInput
		if err := strictUnmarshal(raw, &list); err != nil {
			return nil, 0, "", err
		}
		return list, 0, "", nil
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, 0, "", err
	}
	if _, ok := probe["components"]; ok {
		var request struct {
			Components      []application.ComponentInput `json:"components"`
			ExpectedVersion uint64                       `json:"expectedVersion"`
			Actor           string                       `json:"actor"`
		}
		if err := strictUnmarshal(raw, &request); err != nil {
			return nil, 0, "", err
		}
		return request.Components, request.ExpectedVersion, request.Actor, nil
	}
	var request struct {
		ComponentCode      string   `json:"componentCode"`
		ComponentType      string   `json:"componentType"`
		Location           string   `json:"location"`
		LoadPathParentCode string   `json:"loadPathParentCode"`
		BaselineNote       string   `json:"baselineNote"`
		RequiredChecks     []string `json:"requiredChecks"`
		ExpectedVersion    uint64   `json:"expectedVersion"`
		Actor              string   `json:"actor"`
	}
	if err := strictUnmarshal(raw, &request); err != nil {
		return nil, 0, "", err
	}
	return []application.ComponentInput{{ComponentCode: request.ComponentCode, ComponentType: request.ComponentType, Location: request.Location, LoadPathParentCode: request.LoadPathParentCode, BaselineNote: request.BaselineNote, RequiredChecks: request.RequiredChecks}}, request.ExpectedVersion, request.Actor, nil
}

func (a *API) observations(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var in struct {
		ComponentID, ConditionType, LocationDetail string
		Severity                                   domain.Severity
		Measurements                               map[string]float64
		EvidenceRefs                               []string
		ObservedAt                                 time.Time
		ExpectedVersion                            uint64
		Actor                                      string
	}
	if e := decode(r, &in); e != nil {
		errWrite(w, e)
		return
	}
	d, e := a.s.Observe(id, application.ObservationInput{ComponentID: in.ComponentID, ConditionType: in.ConditionType, LocationDetail: in.LocationDetail, Severity: in.Severity, Measurements: in.Measurements, EvidenceRefs: in.EvidenceRefs, ObservedAt: in.ObservedAt}, requestVersion(r, in.ExpectedVersion), requestActor(r, in.Actor), key(r))
	if e != nil {
		errWrite(w, e)
		return
	}
	write(w, 200, d)
}
func (a *API) assess(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var in struct {
		ExpectedVersion uint64 `json:"expectedVersion"`
		Actor           string `json:"actor"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if e := decode(r, &in); e != nil {
			errWrite(w, e)
			return
		}
	}
	res, d, e := a.s.AssessKey(id, requestVersion(r, in.ExpectedVersion), requestActor(r, in.Actor), key(r))
	if e != nil {
		errWrite(w, e)
		return
	}
	write(w, 200, map[string]any{"dossier": d, "assessment": res})
}
func (a *API) plans(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var in struct {
		application.PlanInput
		ExpectedVersion uint64 `json:"expectedVersion"`
		Actor           string `json:"actor"`
	}
	if e := decode(r, &in); e != nil {
		errWrite(w, e)
		return
	}
	d, e := a.s.SubmitPlanKey(id, in.PlanInput, requestVersion(r, in.ExpectedVersion), requestActor(r, in.Actor), key(r))
	if e != nil {
		errWrite(w, e)
		return
	}
	write(w, 200, d)
}
func (a *API) reviews(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var in struct {
		application.ReviewInput
		ExpectedVersion uint64 `json:"expectedVersion"`
		Actor           string `json:"actor"`
	}
	if e := decode(r, &in); e != nil {
		errWrite(w, e)
		return
	}
	d, e := a.s.ReviewKey(id, in.Decision, in.Comments, in.ResolvedFindingIDs, requestVersion(r, in.ExpectedVersion), requestActor(r, in.Actor), key(r))
	if e != nil {
		errWrite(w, e)
		return
	}
	write(w, 200, d)
}
func (a *API) freeze(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var in struct {
		ManifestHash    string `json:"manifestHash"`
		ExpectedVersion uint64 `json:"expectedVersion"`
		Actor           string `json:"actor"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if e := decode(r, &in); e != nil {
			errWrite(w, e)
			return
		}
	}
	d, e := a.s.FreezeWithHashKey(id, in.ManifestHash, requestVersion(r, in.ExpectedVersion), requestActor(r, in.Actor), key(r))
	if e != nil {
		errWrite(w, e)
		return
	}
	write(w, 200, d)
}
func (a *API) release(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		method(w)
		return
	}
	var in struct {
		ExpectedVersion uint64 `json:"expectedVersion"`
		Actor           string `json:"actor"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if e := decode(r, &in); e != nil {
			errWrite(w, e)
			return
		}
	}
	d, e := a.s.ReleaseKey(id, requestActor(r, in.Actor), requestVersion(r, in.ExpectedVersion), key(r))
	if e != nil {
		errWrite(w, e)
		return
	}
	write(w, 200, d.Certificate)
}

func requestVersion(r *http.Request, body uint64) uint64 {
	if value := expected(r); value != 0 {
		return value
	}
	return body
}

func requestActor(r *http.Request, body string) string {
	if value := r.Header.Get("X-Actor"); value != "" {
		return value
	}
	if body != "" {
		return body
	}
	return "anonymous"
}
