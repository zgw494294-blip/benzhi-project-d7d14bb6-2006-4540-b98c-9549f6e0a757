package store

// RecoverSnapshot validates the projection against the verified ledger and
// atomically rebuilds it from ledger projections when it is missing or stale.
func (s *Store) RecoverSnapshot() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.validateOrRecoverSnapshot()
}
