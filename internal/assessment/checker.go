package assessment

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"timber-release-gate/internal/domain"
)

type Result struct {
	Findings []domain.ReviewFinding `json:"findings"`
	Changes  []domain.FindingDelta  `json:"changes"`
	Risk     string                 `json:"risk"`
	Complete bool                   `json:"complete"`
}

func Run(d *domain.SurveyDossier) Result {
	var fs []domain.ReviewFinding
	for id, c := range d.Components {
		o, ok := d.LatestObservation(id)
		if !ok {
			fs = append(fs, finding(d, "MISSING_OBSERVATION", []string{id}, domain.SeverityHigh, "构件缺少病害观察", true))
			continue
		}
		if len(o.EvidenceRefs) == 0 {
			fs = append(fs, finding(d, "MISSING_EVIDENCE", []string{id}, domain.SeverityHigh, "观察缺少证据引用", true))
		}
		if observationConflicts(o) {
			fs = append(fs, finding(d, "CONFLICTING_OBSERVATION", []string{id}, domain.SeverityHigh, "病害结论、严重度与测量值相互冲突", true))
		}
		if invalidMeasurement(o) {
			fs = append(fs, finding(d, "MEASUREMENT_OUT_OF_RANGE", []string{id}, domain.SeverityCritical, "观察包含超出允许边界的测量值", true))
		}
		if domain.SeverityRank(o.Severity) >= domain.SeverityRank(domain.SeverityHigh) {
			fs = append(fs, finding(d, "HIGH_RISK", []string{id}, o.Severity, fmt.Sprintf("构件%s存在高风险病害", c.ComponentCode), true))
		}
		if c.LoadPathParentCode != "" {
			parentID := ""
			for candidateID, pc := range d.Components {
				if pc.ComponentCode == c.LoadPathParentCode {
					parentID = candidateID
					break
				}
			}
			if parentID == "" {
				fs = append(fs, finding(d, "LOAD_PATH_GAP", []string{id}, domain.SeverityCritical, "承重传递链缺少上级构件", true))
			} else if _, observed := d.LatestObservation(parentID); !observed {
				fs = append(fs, finding(d, "LOAD_PATH_CHECK_GAP", []string{parentID, id}, domain.SeverityHigh, "承重传递链上级构件尚未完成观察", true))
			}
		}
	}
	sort.Slice(fs, func(i, j int) bool {
		if domain.SeverityRank(fs[i].Severity) != domain.SeverityRank(fs[j].Severity) {
			return domain.SeverityRank(fs[i].Severity) > domain.SeverityRank(fs[j].Severity)
		}
		if fs[i].Code != fs[j].Code {
			return fs[i].Code < fs[j].Code
		}
		ci, cj := findingComponentCode(d, fs[i]), findingComponentCode(d, fs[j])
		if ci != cj {
			return ci < cj
		}
		return fs[i].ID < fs[j].ID
	})
	risk := "LOW"
	for _, f := range fs {
		if f.Severity == domain.SeverityCritical {
			risk = "CRITICAL"
			break
		}
		if f.Severity == domain.SeverityHigh {
			risk = "HIGH"
		} else if f.Severity == domain.SeverityMedium && risk == "LOW" {
			risk = "MEDIUM"
		}
	}
	complete := true
	for _, f := range fs {
		if f.Code == "MISSING_OBSERVATION" || f.Code == "MISSING_EVIDENCE" || f.Code == "LOAD_PATH_GAP" || f.Code == "LOAD_PATH_CHECK_GAP" || f.Code == "MEASUREMENT_OUT_OF_RANGE" {
			complete = false
		}
	}
	return Result{Findings: fs, Risk: risk, Complete: complete}
}

func observationConflicts(observation domain.ConditionObservation) bool {
	condition := strings.ToLower(strings.TrimSpace(observation.ConditionType))
	declaresHealthy := strings.Contains(condition, "无病害") || strings.Contains(condition, "未见病害") || strings.Contains(condition, "完好")
	if !declaresHealthy {
		return false
	}
	if domain.SeverityRank(observation.Severity) >= domain.SeverityRank(domain.SeverityHigh) {
		return true
	}
	for _, value := range observation.Measurements {
		if value > 0 {
			return true
		}
	}
	return false
}

func invalidMeasurement(observation domain.ConditionObservation) bool {
	for key, value := range observation.Measurements {
		limit, supported := domain.MeasurementLimits[key]
		if !supported || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > limit {
			return true
		}
	}
	return false
}
func finding(d *domain.SurveyDossier, code string, cs []string, s domain.Severity, msg string, b bool) domain.ReviewFinding {
	return domain.ReviewFinding{ID: domain.Digest(struct{ D, C string }{d.ID, code + fmt.Sprint(cs)})[:20], DossierID: d.ID, Source: domain.SourceAutomatic, Code: code, ComponentIDs: cs, Severity: s, Message: msg, Blocking: b, Status: domain.FindingOpen}
}

func findingComponentCode(d *domain.SurveyDossier, f domain.ReviewFinding) string {
	if len(f.ComponentIDs) == 0 {
		return ""
	}
	return d.Components[f.ComponentIDs[0]].ComponentCode
}
