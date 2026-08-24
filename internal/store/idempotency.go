package store

import "timber-release-gate/internal/domain"

func (s *Store) Idempotent(key, hash string) (*domain.SurveyDossier, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.idem[key]
	if !ok {
		return nil, false, nil
	}
	if r.RequestHash != hash {
		err := &domain.Error{Code: "IDEMPOTENCY_CONFLICT", Message: "幂等键对应请求不同"}
		if r.Dossier != nil {
			err.State = r.Dossier.State
			err.Version = r.Dossier.Version
		}
		return nil, true, err
	}
	return domain.CloneDossier(r.Dossier), true, nil
}
