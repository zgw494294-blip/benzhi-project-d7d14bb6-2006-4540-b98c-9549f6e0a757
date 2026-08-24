package domain

import "sort"

func AuditHash(e AuditEvent) string {
	e.Hash = ""
	return Digest(e)
}

func VerifyAudit(events []AuditEvent) bool {
	prev := ""
	for i, e := range events {
		if e.Sequence != uint64(i+1) || e.PreviousHash != prev || e.Hash != AuditHash(e) {
			return false
		}
		prev = e.Hash
	}
	return true
}
func SortEvents(events []AuditEvent) []AuditEvent {
	out := append([]AuditEvent(nil), events...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out
}
func EventSummary(events []AuditEvent) map[string]int {
	out := map[string]int{}
	for _, e := range events {
		out[e.EventType]++
	}
	return out
}
func LastEvent(events []AuditEvent) (AuditEvent, bool) {
	if len(events) == 0 {
		return AuditEvent{}, false
	}
	return events[len(events)-1], true
}
