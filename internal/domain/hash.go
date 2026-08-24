package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

func Digest(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (d *SurveyDossier) FreezeHash(plan RepairPlanRevision) string {
	ids := make([]string, 0, len(d.Components))
	for id := range d.Components {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	obs := d.LatestObservationIDs()
	sort.Strings(obs)
	m := FreezeManifest{DossierID: d.ID, DossierVersion: d.Version, ComponentIDs: ids, ObservationIDs: obs, PlanID: plan.ID, PlanRevision: plan.Revision}
	return d.manifestDigest(&m, plan)
}

func (m FreezeManifest) Digest() string {
	components := append([]string(nil), m.ComponentIDs...)
	observations := append([]string(nil), m.ObservationIDs...)
	sort.Strings(components)
	sort.Strings(observations)
	return Digest(struct {
		DossierID      string
		DossierVersion uint64
		ComponentIDs   []string
		ObservationIDs []string
		PlanID         string
		PlanRevision   uint32
	}{m.DossierID, m.DossierVersion, components, observations, m.PlanID, m.PlanRevision})
}
func (c WorkReleaseCertificate) Digest() string {
	return Digest(struct {
		ID, DossierID, ManifestHash, AuditHash, IssuedBy string
		DossierVersion                                   uint64
		IssuedAt                                         string
	}{c.ID, c.DossierID, c.FreezeManifestHash, c.AuditHeadHash, c.IssuedBy, c.DossierVersion, c.IssuedAt.UTC().Format(time.RFC3339Nano)})
}
