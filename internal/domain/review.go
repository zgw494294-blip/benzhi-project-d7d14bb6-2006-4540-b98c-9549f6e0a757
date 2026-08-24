package domain

import (
	"fmt"
	"strings"
)

func (d *SurveyDossier) Review(decision ReviewDecision, comments []string, resolved []string) error {
	if err := d.Mutable(); err != nil {
		return err
	}
	if d.State != StatePlanSubmitted {
		return d.err("INVALID_STATE", "当前状态不可审核")
	}
	p, ok := d.CurrentPlan()
	if !ok {
		return d.err("PLAN_MISSING", "没有可审核方案")
	}
	if decision != DecisionReturn && decision != DecisionApprove {
		return d.err("INVALID_DECISION", "审核决定只允许RETURN或APPROVE")
	}
	cleanComments := normalizeStrings(comments)
	if decision == DecisionReturn && len(cleanComments) == 0 {
		return d.err("REVIEW_COMMENT_REQUIRED", "退回方案必须至少提供一条逐项意见")
	}
	resolved = uniqueStrings(append(append([]string(nil), p.ResolvedFindingIDs...), resolved...))
	resolvedSet := make(map[string]bool, len(resolved))
	for _, id := range resolved {
		f, yes := d.Findings[id]
		if !yes {
			return d.err("UNKNOWN_FINDING", "解决标识引用了不存在的问题项")
		}
		if !f.Blocking {
			return d.err("NON_BLOCKING_FINDING", "解决标识只能引用阻断问题")
		}
		resolvedSet[id] = true
	}
	if decision == DecisionApprove {
		for _, f := range d.Findings {
			if f.Blocking && f.Status != FindingResolved && !resolvedSet[f.ID] {
				return d.err("BLOCKING_FINDING", "存在未解决阻断意见")
			}
		}
	}
	for id := range resolvedSet {
		f := d.Findings[id]
		f.Status = FindingResolved
		f.ResolvedByRevision = p.Revision
		d.Findings[id] = f
	}
	p.ReviewComments = append([]string(nil), cleanComments...)
	p.Decision = decision
	if decision == DecisionReturn {
		for index, comment := range cleanComments {
			id := Digest(struct {
				DossierID string
				Revision  uint32
				Index     int
				Comment   string
			}{d.ID, p.Revision, index, comment})[:20]
			d.Findings[id] = ReviewFinding{
				ID: id, DossierID: d.ID, Source: SourceReview,
				Code: fmt.Sprintf("REVIEW_COMMENT_%d", index+1), Severity: SeverityHigh,
				Message: strings.TrimSpace(comment), Blocking: true, Status: FindingOpen,
			}
		}
		if err := d.Transition(StateChangesRequested); err != nil {
			return err
		}
	} else if err := d.Transition(StateApproved); err != nil {
		return err
	}
	d.bump()
	return nil
}
