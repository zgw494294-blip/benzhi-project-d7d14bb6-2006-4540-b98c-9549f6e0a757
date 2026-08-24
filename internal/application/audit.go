package application

import "timber-release-gate/internal/domain"

func EventTypes() []string {
	return []string{"DOSSIER_CREATED", "COMPONENT_ADDED", "OBSERVATION_ADDED", "ASSESSED", "PLAN_SUBMITTED", "REVIEW_RECORDED", "DOSSIER_FROZEN", "RELEASE_ISSUED"}
}
func IsTerminal(state domain.DossierState) bool { return state == domain.StateReleased }
