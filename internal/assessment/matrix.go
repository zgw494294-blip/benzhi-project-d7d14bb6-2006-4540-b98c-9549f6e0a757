package assessment

import (
	"sort"
	"timber-release-gate/internal/domain"
)

type MatrixRow struct {
	ComponentID   string          `json:"componentID"`
	ComponentCode string          `json:"componentCode"`
	Severity      domain.Severity `json:"severity"`
	FindingIDs    []string        `json:"findingIDs"`
	Covered       bool            `json:"covered"`
}

func BuildMatrix(d *domain.SurveyDossier) []MatrixRow {
	rows := []MatrixRow{}
	for id, c := range d.Components {
		r := MatrixRow{ComponentID: id, ComponentCode: c.ComponentCode}
		if o, ok := d.LatestObservation(id); ok {
			r.Severity = o.Severity
		}
		for fid, f := range d.Findings {
			for _, cid := range f.ComponentIDs {
				if cid == id {
					r.FindingIDs = append(r.FindingIDs, fid)
				}
			}
		}
		if p, ok := d.CurrentPlan(); ok {
			r.Covered = d.PlanCoversComponent(*p, id)
		}
		sort.Strings(r.FindingIDs)
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ComponentCode < rows[j].ComponentCode })
	return rows
}
