package application

import (
	"sort"
	"timber-release-gate/internal/domain"
)

type ObservationHistoryItem struct {
	domain.ConditionObservation
	ComponentCode string `json:"componentCode"`
	Current       bool   `json:"current"`
}

type PlanRevisionDiff struct {
	FromRevision               uint32                `json:"fromRevision"`
	ToRevision                 uint32                `json:"toRevision"`
	AddedActions               []domain.RepairAction `json:"addedActions"`
	RemovedActions             []domain.RepairAction `json:"removedActions"`
	ChangedActions             []domain.RepairAction `json:"changedActions"`
	AddedObservationIDs        []string              `json:"addedObservationIDs"`
	RemovedObservationIDs      []string              `json:"removedObservationIDs"`
	MaterialConstraintsChanged bool                  `json:"materialConstraintsChanged"`
	AcceptanceCriteriaChanged  bool                  `json:"acceptanceCriteriaChanged"`
}

func (s *Service) View(id string) (map[string]any, error) {
	d, ok := s.st.Get(id)
	if !ok {
		return nil, &domain.Error{Code: "NOT_FOUND", Message: "档案不存在"}
	}
	tl, err := s.Timeline(id)
	if err != nil {
		return nil, err
	}
	risk, err := s.QueryRisk(id, RiskFilter{})
	if err != nil {
		return nil, err
	}
	release, _ := s.ReleaseView(id)
	return map[string]any{"dossier": d, "observationHistory": buildObservationHistory(d), "planDiffs": buildPlanDiffs(d.Plans), "timeline": tl, "risk": risk, "release": release}, nil
}

func buildObservationHistory(d *domain.SurveyDossier) []ObservationHistoryItem {
	out := []ObservationHistoryItem{}
	for componentID, observations := range d.Observations {
		componentCode := d.Components[componentID].ComponentCode
		for i, observation := range observations {
			out = append(out, ObservationHistoryItem{ConditionObservation: observation, ComponentCode: componentCode, Current: i == len(observations)-1})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ComponentCode != out[j].ComponentCode {
			return out[i].ComponentCode < out[j].ComponentCode
		}
		return out[i].Revision < out[j].Revision
	})
	return out
}

func buildPlanDiffs(plans []domain.RepairPlanRevision) []PlanRevisionDiff {
	out := []PlanRevisionDiff{}
	for i := 1; i < len(plans); i++ {
		before, after := plans[i-1], plans[i]
		diff := PlanRevisionDiff{FromRevision: before.Revision, ToRevision: after.Revision, AddedActions: []domain.RepairAction{}, RemovedActions: []domain.RepairAction{}, ChangedActions: []domain.RepairAction{}}
		beforeActions, afterActions := actionIndex(before.Actions), actionIndex(after.Actions)
		for key, action := range afterActions {
			old, exists := beforeActions[key]
			if !exists {
				diff.AddedActions = append(diff.AddedActions, action)
			} else if domain.Digest(old) != domain.Digest(action) {
				diff.ChangedActions = append(diff.ChangedActions, action)
			}
		}
		for key, action := range beforeActions {
			if _, exists := afterActions[key]; !exists {
				diff.RemovedActions = append(diff.RemovedActions, action)
			}
		}
		diff.AddedObservationIDs, diff.RemovedObservationIDs = stringSetDiff(before.ReferencedObservationIDs, after.ReferencedObservationIDs)
		diff.MaterialConstraintsChanged = domain.Digest(before.MaterialConstraints) != domain.Digest(after.MaterialConstraints)
		diff.AcceptanceCriteriaChanged = domain.Digest(before.AcceptanceCriteria) != domain.Digest(after.AcceptanceCriteria)
		sortActions(diff.AddedActions)
		sortActions(diff.RemovedActions)
		sortActions(diff.ChangedActions)
		out = append(out, diff)
	}
	return out
}

func actionIndex(actions []domain.RepairAction) map[string]domain.RepairAction {
	out := make(map[string]domain.RepairAction, len(actions))
	for _, action := range actions {
		out[action.FindingID+"|"+action.ComponentID] = action
	}
	return out
}

func sortActions(actions []domain.RepairAction) {
	sort.Slice(actions, func(i, j int) bool {
		ki := actions[i].FindingID + "|" + actions[i].ComponentID
		kj := actions[j].FindingID + "|" + actions[j].ComponentID
		return ki < kj
	})
}

func stringSetDiff(before, after []string) (added, removed []string) {
	b, a := map[string]bool{}, map[string]bool{}
	for _, value := range before {
		b[value] = true
	}
	for _, value := range after {
		a[value] = true
	}
	for value := range a {
		if !b[value] {
			added = append(added, value)
		}
	}
	for value := range b {
		if !a[value] {
			removed = append(removed, value)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}
