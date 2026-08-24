package application

import (
	"sort"
	"timber-release-gate/internal/domain"
)

type ReleaseView struct {
	DossierID     string                         `json:"dossierID"`
	State         domain.DossierState            `json:"state"`
	Version       uint64                         `json:"version"`
	RiskItems     []map[string]any               `json:"riskItems"`
	Certificate   *domain.WorkReleaseCertificate `json:"certificate,omitempty"`
	Manifest      *domain.FreezeManifest         `json:"manifest,omitempty"`
	Timeline      domain.Timeline                `json:"timeline"`
	PlanDiffs     []PlanRevisionDiff             `json:"planDiffs"`
	AuditVerified bool                           `json:"auditVerified"`
	Released      bool                           `json:"released"`
}

func (s *Service) ReleaseView(id string) (ReleaseView, error) {
	d, ok := s.st.Get(id)
	if !ok {
		return ReleaseView{}, domain.Invalid("NOT_FOUND", "档案不存在")
	}
	risk, _ := s.Risk(id)
	sort.SliceStable(risk, func(i, j int) bool {
		if risk[i]["findingID"].(string) != risk[j]["findingID"].(string) {
			return risk[i]["findingID"].(string) < risk[j]["findingID"].(string)
		}
		return risk[i]["componentCode"].(string) < risk[j]["componentCode"].(string)
	})
	timeline, _ := s.st.Timeline(id)
	verified := s.st.VerifyChain() == nil
	if d.Certificate != nil && validateReleaseCertificate(d, timeline.Events) != nil {
		verified = false
	}
	return ReleaseView{DossierID: d.ID, State: d.State, Version: d.Version, RiskItems: risk, Certificate: d.Certificate, Manifest: d.Manifest, Timeline: timeline, PlanDiffs: buildPlanDiffs(d.Plans), AuditVerified: verified, Released: d.State == domain.StateReleased && d.Certificate != nil && verified}, nil
}
func (s *Service) AuditEvents(id string) []domain.AuditEvent { return s.st.Events(id) }
func (s *Service) CurrentVersion(id string) (uint64, error) {
	v, ok := s.st.SnapshotVersion(id)
	if !ok {
		return 0, domain.Invalid("NOT_FOUND", "档案不存在")
	}
	return v, nil
}
