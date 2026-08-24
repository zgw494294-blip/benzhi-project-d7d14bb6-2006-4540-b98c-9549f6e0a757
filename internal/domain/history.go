package domain

import (
	"sort"
	"time"
)

type RevisionSummary struct {
	ComponentID   string    `json:"componentID"`
	Revision      uint32    `json:"revision"`
	ObservationID string    `json:"observationID"`
	Severity      Severity  `json:"severity"`
	ObservedAt    time.Time `json:"observedAt"`
}

func (d *SurveyDossier) ObservationHistory(componentID string) []RevisionSummary {
	xs := d.Observations[componentID]
	out := make([]RevisionSummary, 0, len(xs))
	for _, o := range xs {
		out = append(out, RevisionSummary{componentID, o.Revision, o.ID, o.Severity, o.ObservedAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Revision < out[j].Revision })
	return out
}
func (d *SurveyDossier) AllObservations() []ConditionObservation {
	out := []ConditionObservation{}
	for _, xs := range d.Observations {
		out = append(out, xs...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ComponentID != out[j].ComponentID {
			return out[i].ComponentID < out[j].ComponentID
		}
		return out[i].Revision < out[j].Revision
	})
	return out
}
func (d *SurveyDossier) LatestObservationIDs() []string {
	out := []string{}
	for id := range d.Components {
		if o, ok := d.LatestObservation(id); ok {
			out = append(out, o.ID)
		}
	}
	sort.Strings(out)
	return out
}
func (d *SurveyDossier) FindingCodes() []string {
	out := []string{}
	for _, f := range d.Findings {
		out = append(out, f.Code)
	}
	sort.Strings(out)
	return out
}
func (d *SurveyDossier) StateHistory(events []AuditEvent) []DossierState {
	out := []DossierState{StateDraft}
	for _, e := range events {
		switch e.EventType {
		case "COMPONENT_ADDED":
			out = append(out, StateSurveying)
		case "ASSESSED":
			out = append(out, StateAssessed)
		case "PLAN_SUBMITTED":
			out = append(out, StatePlanSubmitted)
		case "REVIEW_RECORDED":
			if e.State != "" {
				out = append(out, e.State)
			} else {
				// 兼容未记录状态的旧审核事件。
				out = append(out, StateApproved)
			}
		case "DOSSIER_FROZEN":
			out = append(out, StateFrozen)
		case "RELEASE_ISSUED":
			out = append(out, StateReleased)
		}
	}
	return out
}
