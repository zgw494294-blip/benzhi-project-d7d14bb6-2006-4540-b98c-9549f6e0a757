package domain

import (
	"sort"
	"strings"
)

func (d *SurveyDossier) SubmitPlan(p RepairPlanRevision) error {
	if err := d.Mutable(); err != nil {
		return err
	}
	if d.State != StateAssessed && d.State != StateChangesRequested {
		return d.err("INVALID_STATE", "只有评估完成或退回状态可提交方案")
	}
	if len(p.Actions) == 0 {
		return d.err("PLAN_INCOMPLETE", "方案动作不能为空")
	}
	max := uint32(0)
	for _, x := range d.Plans {
		if x.Revision > max {
			max = x.Revision
		}
	}
	if p.Revision <= max {
		return d.err("REVISION_CONFLICT", "方案修订必须递增")
	}
	allObservations := map[string]ConditionObservation{}
	latestObservations := map[string]ConditionObservation{}
	for componentID, observations := range d.Observations {
		for _, observation := range observations {
			allObservations[observation.ID] = observation
		}
		if len(observations) > 0 {
			latestObservations[componentID] = observations[len(observations)-1]
		}
	}
	referenced := map[string]bool{}
	for _, id := range p.ReferencedObservationIDs {
		observation, ok := allObservations[id]
		if !ok {
			return d.err("UNKNOWN_OBSERVATION", "方案引用了不属于档案的观察")
		}
		if latestObservations[observation.ComponentID].ID != id {
			return d.err("STALE_OBSERVATION_REFERENCE", "方案只能引用构件的最新观察修订")
		}
		referenced[id] = true
	}
	for _, observation := range latestObservations {
		if !referenced[observation.ID] {
			return d.err("OBSERVATION_REFERENCE_REQUIRED", "方案缺少构件最新观察引用")
		}
	}
	covered := map[string]bool{}
	for _, a := range p.Actions {
		if a.ComponentID == "" && a.FindingID == "" {
			return d.err("PLAN_INCOMPLETE", "每个动作必须关联问题项或构件")
		}
		if a.ComponentID != "" {
			if _, ok := d.Components[a.ComponentID]; !ok {
				return d.err("UNKNOWN_COMPONENT", "方案动作引用了未知构件")
			}
			covered[a.ComponentID] = true
		}
		if a.FindingID != "" {
			finding, ok := d.Findings[a.FindingID]
			if !ok {
				return d.err("UNKNOWN_FINDING", "方案动作引用了未知问题项")
			}
			for _, componentID := range finding.ComponentIDs {
				covered[componentID] = true
			}
		}
		if strings.TrimSpace(a.Action) == "" || strings.TrimSpace(a.MaterialConstraint) == "" || strings.TrimSpace(a.AcceptanceStandard) == "" {
			return d.err("PLAN_INCOMPLETE", "每个动作必须包含处置、材料约束和验收标准")
		}
	}
	for _, f := range d.Findings {
		if f.Blocking && f.Status != FindingResolved && !coveredFinding(p, f) {
			return d.err("FINDING_UNRESOLVED", "方案未覆盖阻断问题")
		}
	}
	for _, id := range p.ResolvedFindingIDs {
		if _, ok := d.Findings[id]; !ok {
			return d.err("UNKNOWN_FINDING", "方案声明解决了未知问题项")
		}
	}
	for _, c := range d.Components {
		if o, ok := d.LatestObservation(c.ID); ok && SeverityRank(o.Severity) >= SeverityRank(SeverityHigh) && !covered[c.ID] {
			return d.err("HIGH_RISK_UNCOVERED", "高风险构件未覆盖")
		}
	}
	p.ID = Digest(struct {
		D string
		R uint32
	}{d.ID, p.Revision})[:20]
	p.DossierID = d.ID
	p.Decision = DecisionPending
	p.Actions = append([]RepairAction(nil), p.Actions...)
	p.ReferencedObservationIDs = uniqueStrings(p.ReferencedObservationIDs)
	p.ResolvedFindingIDs = uniqueStrings(p.ResolvedFindingIDs)
	p.MaterialConstraints = append([]string(nil), p.MaterialConstraints...)
	p.AcceptanceCriteria = append([]string(nil), p.AcceptanceCriteria...)
	p.ReviewComments = nil
	d.Plans = append(d.Plans, p)
	if err := d.Transition(StatePlanSubmitted); err != nil {
		return err
	}
	d.bump()
	return nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
func coveredFinding(p RepairPlanRevision, finding ReviewFinding) bool {
	for _, x := range p.ResolvedFindingIDs {
		if x == finding.ID {
			return true
		}
	}
	for _, a := range p.Actions {
		if a.FindingID == finding.ID {
			return true
		}
		for _, componentID := range finding.ComponentIDs {
			if a.ComponentID == componentID {
				return true
			}
		}
	}
	return false
}
func (d *SurveyDossier) CurrentPlan() (*RepairPlanRevision, bool) {
	if len(d.Plans) == 0 {
		return nil, false
	}
	return &d.Plans[len(d.Plans)-1], true
}
func SortFindings(fs []ReviewFinding) {
	sort.Slice(fs, func(i, j int) bool {
		if SeverityRank(fs[i].Severity) != SeverityRank(fs[j].Severity) {
			return SeverityRank(fs[i].Severity) > SeverityRank(fs[j].Severity)
		}
		if fs[i].Code != fs[j].Code {
			return fs[i].Code < fs[j].Code
		}
		return fs[i].ID < fs[j].ID
	})
}

func SortFindingDeltas(ds []FindingDelta) {
	sort.SliceStable(ds, func(i, j int) bool {
		if SeverityRank(ds[i].Finding.Severity) != SeverityRank(ds[j].Finding.Severity) {
			return SeverityRank(ds[i].Finding.Severity) > SeverityRank(ds[j].Finding.Severity)
		}
		if ds[i].Finding.Code != ds[j].Finding.Code {
			return ds[i].Finding.Code < ds[j].Finding.Code
		}
		return ds[i].Finding.ID < ds[j].Finding.ID
	})
}
