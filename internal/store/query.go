package store

import "timber-release-gate/internal/domain"

func (s *Store) Timeline(id string) (domain.Timeline, error) {
	d, ok := s.Get(id)
	if !ok {
		return domain.Timeline{}, &domain.Error{Code: "NOT_FOUND", Message: "档案不存在"}
	}
	return domain.Timeline{Events: s.Events(id), CurrentVersion: d.Version, State: d.State}, nil
}
