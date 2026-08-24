package domain

func (d *SurveyDossier) Transition(next DossierState) error {
	allowed := map[DossierState]map[DossierState]bool{StateDraft: {StateSurveying: true}, StateSurveying: {StateAssessed: true}, StateAssessed: {StatePlanSubmitted: true}, StatePlanSubmitted: {StateChangesRequested: true, StateApproved: true}, StateChangesRequested: {StatePlanSubmitted: true}, StateApproved: {StateFrozen: true}, StateFrozen: {StateReleased: true}}
	if !allowed[d.State][next] {
		return d.err("INVALID_STATE", "当前状态不允许该操作")
	}
	d.State = next
	return nil
}
func (d *SurveyDossier) Mutable() error {
	if d.State == StateFrozen || d.State == StateReleased {
		return d.err("FROZEN", "档案已冻结，禁止修改业务内容")
	}
	return nil
}
