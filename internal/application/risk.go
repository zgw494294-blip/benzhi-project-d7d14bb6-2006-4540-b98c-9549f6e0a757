package application

import (
	"sort"
	"timber-release-gate/internal/domain"
)

type RiskFilter struct {
	ComponentCode string
	Severity      domain.Severity
	Status        domain.FindingStatus
	Covered       *bool
}

type RiskItem struct {
	FindingID     string               `json:"findingID"`
	Code          string               `json:"code"`
	ComponentID   string               `json:"componentID,omitempty"`
	ComponentCode string               `json:"componentCode,omitempty"`
	Severity      domain.Severity      `json:"severity"`
	Blocking      bool                 `json:"blocking"`
	Status        domain.FindingStatus `json:"status"`
	Covered       bool                 `json:"covered"`
	Message       string               `json:"message"`
}

type RiskSummary struct {
	TotalComponents     int    `json:"totalComponents"`
	HighRiskComponents  int    `json:"highRiskComponents"`
	CoveredCount        int    `json:"coveredCount"`
	OpenBlockingCount   int    `json:"openBlockingCount"`
	CurrentPlanRevision uint32 `json:"currentPlanRevision"`
}

type RiskResponse struct {
	Items         []RiskItem  `json:"items"`
	Summary       RiskSummary `json:"summary"`
	AuditVerified bool        `json:"auditVerified"`
}

func (s *Service) QueryRisk(id string, filter RiskFilter) (RiskResponse, error) {
	if err := s.st.VerifyChain(); err != nil {
		return RiskResponse{}, &domain.Error{Code: "AUDIT_INTEGRITY_ERROR", Message: err.Error()}
	}
	d, ok := s.st.Get(id)
	if !ok {
		return RiskResponse{}, &domain.Error{Code: "NOT_FOUND", Message: "档案不存在"}
	}
	if filter.Severity != "" && domain.SeverityRank(filter.Severity) == 0 {
		return RiskResponse{}, &domain.Error{Code: "INVALID_FILTER", Message: "severity筛选值无效"}
	}
	if filter.Status != "" && filter.Status != domain.FindingOpen && filter.Status != domain.FindingResolved {
		return RiskResponse{}, &domain.Error{Code: "INVALID_FILTER", Message: "status筛选值无效"}
	}
	cacheable := filter.ComponentCode == "" && filter.Severity == "" && filter.Status == "" && filter.Covered == nil
	if cacheable {
		if response, exists := s.cachedRisk(id); exists {
			return response, nil
		}
	}
	var plan *domain.RepairPlanRevision
	if current, exists := d.CurrentPlan(); exists {
		plan = current
	}
	coveredComponent := func(id string) bool { return plan != nil && d.PlanCoversComponent(*plan, id) }
	response := RiskResponse{Items: []RiskItem{}, Summary: RiskSummary{TotalComponents: len(d.Components)}, AuditVerified: true}
	if plan != nil {
		response.Summary.CurrentPlanRevision = plan.Revision
	}
	for id := range d.Components {
		if observation, exists := d.LatestObservation(id); exists && domain.SeverityRank(observation.Severity) >= domain.SeverityRank(domain.SeverityHigh) {
			response.Summary.HighRiskComponents++
		}
		if coveredComponent(id) {
			response.Summary.CoveredCount++
		}
	}
	for _, finding := range d.Findings {
		if finding.Blocking && finding.Status == domain.FindingOpen {
			response.Summary.OpenBlockingCount++
		}
		componentIDs := append([]string(nil), finding.ComponentIDs...)
		if len(componentIDs) == 0 {
			componentIDs = []string{""}
		}
		for _, componentID := range componentIDs {
			componentCode := ""
			if component, exists := d.Components[componentID]; exists {
				componentCode = component.ComponentCode
			}
			covered := findingCovered(plan, finding, componentID)
			if filter.ComponentCode != "" && filter.ComponentCode != componentCode ||
				filter.Severity != "" && filter.Severity != finding.Severity ||
				filter.Status != "" && filter.Status != finding.Status ||
				filter.Covered != nil && *filter.Covered != covered {
				continue
			}
			response.Items = append(response.Items, RiskItem{FindingID: finding.ID, Code: finding.Code, ComponentID: componentID, ComponentCode: componentCode, Severity: finding.Severity, Blocking: finding.Blocking, Status: finding.Status, Covered: covered, Message: finding.Message})
		}
	}
	sort.Slice(response.Items, func(i, j int) bool {
		if response.Items[i].FindingID != response.Items[j].FindingID {
			return response.Items[i].FindingID < response.Items[j].FindingID
		}
		return response.Items[i].ComponentCode < response.Items[j].ComponentCode
	})
	if cacheable {
		s.rememberRisk(id, response)
	}
	return response, nil
}

func (s *Service) cachedRisk(id string) (RiskResponse, bool) {
	s.riskMu.Lock()
	defer s.riskMu.Unlock()
	response, exists := s.riskCache[id]
	response.Items = append([]RiskItem(nil), response.Items...)
	return response, exists
}

func (s *Service) rememberRisk(id string, response RiskResponse) {
	s.riskMu.Lock()
	defer s.riskMu.Unlock()
	response.Items = append([]RiskItem(nil), response.Items...)
	s.riskCache[id] = response
}

func findingCovered(plan *domain.RepairPlanRevision, finding domain.ReviewFinding, componentID string) bool {
	if plan == nil {
		return false
	}
	for _, action := range plan.Actions {
		if action.FindingID == finding.ID || componentID != "" && action.ComponentID == componentID {
			return true
		}
	}
	return false
}
