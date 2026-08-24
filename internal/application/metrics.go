package application

import "timber-release-gate/internal/domain"

type Metrics struct {
	Components       int `json:"components"`
	Observations     int `json:"observations"`
	Findings         int `json:"findings"`
	BlockingFindings int `json:"blockingFindings"`
	Plans            int `json:"plans"`
}

func (s *Service) Metrics(id string) (Metrics, error) {
	d, ok := s.st.Get(id)
	if !ok {
		return Metrics{}, domain.Invalid("NOT_FOUND", "档案不存在")
	}
	m := Metrics{Components: len(d.Components), Findings: len(d.Findings), Plans: len(d.Plans)}
	for _, xs := range d.Observations {
		m.Observations += len(xs)
	}
	for _, f := range d.Findings {
		if f.Blocking && f.Status != domain.FindingResolved {
			m.BlockingFindings++
		}
	}
	return m, nil
}
func (s *Service) IsReadyForConstruction(id string) bool {
	d, ok := s.st.Get(id)
	return ok && d.State == domain.StateReleased && d.Certificate != nil && d.VerifyManifest()
}
func (s *Service) State(id string) (domain.DossierState, bool) {
	d, ok := s.st.Get(id)
	if !ok {
		return "", false
	}
	return d.State, true
}
