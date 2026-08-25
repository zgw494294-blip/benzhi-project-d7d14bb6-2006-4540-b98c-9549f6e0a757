package domain

// CloneDossier isolates command validation and query responses from the stored aggregate.
func CloneDossier(d *SurveyDossier) *SurveyDossier {
	if d == nil {
		return nil
	}
	out := *d
	return &out
}
