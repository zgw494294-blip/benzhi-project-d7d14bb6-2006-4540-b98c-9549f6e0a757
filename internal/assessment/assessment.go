package assessment

import (
	"sort"
	"timber-release-gate/internal/domain"
)

func Apply(d *domain.SurveyDossier) Result {
	r := Run(d)
	previous := d.Findings
	next := make(map[string]domain.ReviewFinding, len(r.Findings))
	for i := range r.Findings {
		f := r.Findings[i]
		if old, ok := previous[f.ID]; ok {
			f.Status = old.Status
			f.ResolvedByRevision = old.ResolvedByRevision
			r.Changes = append(r.Changes, domain.FindingDelta{Change: domain.FindingUnchanged, Finding: f})
		} else {
			r.Changes = append(r.Changes, domain.FindingDelta{Change: domain.FindingAdded, Finding: f})
		}
		r.Findings[i] = f
		next[f.ID] = f
	}
	for id, old := range previous {
		if _, ok := next[id]; !ok {
			r.Changes = append(r.Changes, domain.FindingDelta{Change: domain.FindingRemoved, Finding: old})
		}
	}
	sort.SliceStable(r.Changes, func(i, j int) bool {
		fi, fj := r.Changes[i].Finding, r.Changes[j].Finding
		if domain.SeverityRank(fi.Severity) != domain.SeverityRank(fj.Severity) {
			return domain.SeverityRank(fi.Severity) > domain.SeverityRank(fj.Severity)
		}
		if fi.Code != fj.Code {
			return fi.Code < fj.Code
		}
		ci, cj := findingComponentCode(d, fi), findingComponentCode(d, fj)
		if ci != cj {
			return ci < cj
		}
		return fi.ID < fj.ID
	})
	_ = d.ApplyAssessment(next, r.Complete)
	return r
}
