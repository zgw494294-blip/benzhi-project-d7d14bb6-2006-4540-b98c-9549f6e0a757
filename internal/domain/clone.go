package domain

import "encoding/json"

// CloneDossier isolates command validation and query responses from the stored aggregate.
func CloneDossier(d *SurveyDossier) *SurveyDossier {
	if d == nil {
		return nil
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil
	}
	var out SurveyDossier
	if json.Unmarshal(b, &out) != nil {
		return nil
	}
	if out.Components == nil {
		out.Components = map[string]TimberComponent{}
	}
	if out.Observations == nil {
		out.Observations = map[string][]ConditionObservation{}
	}
	if out.Findings == nil {
		out.Findings = map[string]ReviewFinding{}
	}
	return &out
}
