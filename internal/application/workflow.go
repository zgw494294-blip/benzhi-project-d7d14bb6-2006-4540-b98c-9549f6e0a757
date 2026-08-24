package application

import "timber-release-gate/internal/domain"

func (s *Service) Reassess(id string, expected uint64, actor string) (assessmentResult domainResult, dossier *domain.SurveyDossier, err error) {
	r, d, e := s.Assess(id, expected, actor)
	return domainResult{Risk: r.Risk, Complete: r.Complete, FindingCount: len(r.Findings)}, d, e
}

type domainResult struct {
	Risk         string
	Complete     bool
	FindingCount int
}
type assessmentResult = domainResult
