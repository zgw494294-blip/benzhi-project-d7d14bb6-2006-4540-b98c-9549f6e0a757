package domain

import (
	"sort"
	"strings"
)

func (d *SurveyDossier) Validate() error {
	if d.ID == "" || d.BuildingCode == "" || d.Title == "" {
		return Invalid("INVALID_DOSSIER", "档案基础字段不完整")
	}
	if d.Version == 0 {
		return Invalid("INVALID_VERSION", "档案版本必须为正")
	}
	seen := map[string]bool{}
	for id, c := range d.Components {
		if id == "" || c.ID != id {
			return Invalid("INVALID_COMPONENT", "构件标识不一致")
		}
		if seen[c.ComponentCode] {
			return Invalid("DUPLICATE_COMPONENT", "构件编号重复")
		}
		seen[c.ComponentCode] = true
	}
	for componentID, observations := range d.Observations {
		if _, ok := d.Components[componentID]; !ok {
			return Invalid("ORPHAN_OBSERVATION", "观察引用未知构件")
		}
		for i, o := range observations {
			if o.Revision != uint32(i+1) {
				return Invalid("OBSERVATION_REVISION_GAP", "观察修订不连续")
			}
		}
	}
	return nil
}

func (d *SurveyDossier) ComponentCodes() []string {
	out := make([]string, 0, len(d.Components))
	for _, c := range d.Components {
		out = append(out, c.ComponentCode)
	}
	sort.Strings(out)
	return out
}
func (d *SurveyDossier) HasEvidence(componentID string) bool {
	o, ok := d.LatestObservation(componentID)
	return ok && len(o.EvidenceRefs) > 0
}
func (d *SurveyDossier) BlockingFindings() []ReviewFinding {
	out := []ReviewFinding{}
	for _, f := range d.Findings {
		if f.Blocking && f.Status != FindingResolved {
			out = append(out, f)
		}
	}
	SortFindings(out)
	return out
}

func (d *SurveyDossier) ApplyAssessment(findings map[string]ReviewFinding, complete bool) error {
	if err := d.Mutable(); err != nil {
		return err
	}
	if d.State != StateSurveying && d.State != StateAssessed {
		return d.err("INVALID_STATE", "只有勘察中或已校核档案可执行校核")
	}
	d.Findings = findings
	if complete && d.State == StateSurveying {
		if err := d.Transition(StateAssessed); err != nil {
			return err
		}
	}
	d.bump()
	return nil
}
func (d *SurveyDossier) PlanCoversComponent(plan RepairPlanRevision, id string) bool {
	for _, a := range plan.Actions {
		if a.ComponentID == id {
			return true
		}
		if a.FindingID != "" {
			if finding, ok := d.Findings[a.FindingID]; ok {
				for _, componentID := range finding.ComponentIDs {
					if componentID == id {
						return true
					}
				}
			}
		}
	}
	return false
}
func (d *SurveyDossier) PlanSummary() map[string]any {
	p, ok := d.CurrentPlan()
	if !ok {
		return map[string]any{"revision": 0, "actions": 0}
	}
	return map[string]any{"revision": p.Revision, "actions": len(p.Actions), "decision": p.Decision, "materials": strings.Join(p.MaterialConstraints, ",")}
}
