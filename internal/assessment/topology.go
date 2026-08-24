package assessment

import "timber-release-gate/internal/domain"

func RiskMatrix(d *domain.SurveyDossier) []map[string]any {
	out := []map[string]any{}
	for _, f := range d.Findings {
		out = append(out, map[string]any{"findingID": f.ID, "code": f.Code, "severity": f.Severity, "blocking": f.Blocking, "status": f.Status, "components": f.ComponentIDs})
	}
	return out
}
