package assessment

import (
	"strings"
	"timber-release-gate/internal/domain"
)

type Rule struct {
	Code        string
	Description string
	Blocking    bool
}

var Rules = []Rule{
	{"MISSING_OBSERVATION", "必检构件必须有最新观察", true},
	{"MISSING_EVIDENCE", "观察必须引用现场证据", true},
	{"CONFLICTING_OBSERVATION", "病害结论不得与严重度或测量值冲突", true},
	{"MEASUREMENT_OUT_OF_RANGE", "测量值必须处于受控边界", true},
	{"HIGH_RISK", "高风险构件必须进入方案", true},
	{"LOAD_PATH_GAP", "承重构件必须存在传递链", true},
	{"LOAD_PATH_CHECK_GAP", "承重传递链关联构件必须完成观察", true},
}

func RuleCatalog() []Rule { return append([]Rule(nil), Rules...) }
func NormalizeEvidence(refs []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range refs {
		r = strings.TrimSpace(r)
		if r != "" && !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}
func IsHighRisk(s domain.Severity) bool {
	return domain.SeverityRank(s) >= domain.SeverityRank(domain.SeverityHigh)
}
