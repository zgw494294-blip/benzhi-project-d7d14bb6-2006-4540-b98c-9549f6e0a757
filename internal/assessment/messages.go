package assessment

import "timber-release-gate/internal/domain"

func FindingMessages(fs []domain.ReviewFinding) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.Code+": "+f.Message)
	}
	return out
}
func BlockingCount(fs []domain.ReviewFinding) int {
	n := 0
	for _, f := range fs {
		if f.Blocking && f.Status != domain.FindingResolved {
			n++
		}
	}
	return n
}
