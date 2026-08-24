package store

import (
	"sort"
	"timber-release-gate/internal/domain"
)

type LedgerSummary struct {
	Count        int    `json:"count"`
	HeadHash     string `json:"headHash"`
	LastSequence uint64 `json:"lastSequence"`
}

func (s *Store) Summary() LedgerSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := LedgerSummary{Count: len(s.events)}
	if len(s.events) > 0 {
		out.HeadHash = s.events[len(s.events)-1].Hash
		out.LastSequence = s.events[len(s.events)-1].Sequence
	}
	return out
}
func (s *Store) EventsByType(id, typ string) []domain.AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []domain.AuditEvent{}
	for _, e := range s.events {
		if e.DossierID == id && e.EventType == typ {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out
}
func (s *Store) DossierIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.dossiers))
	for id := range s.dossiers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func (s *Store) Count() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.dossiers) }
