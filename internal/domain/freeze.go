package domain

import (
	"sort"
	"time"
)

func (d *SurveyDossier) Freeze() error {
	return d.FreezeWithHash("")
}

func (d *SurveyDossier) FreezeWithHash(expectedHash string) error {
	if err := d.Mutable(); err != nil {
		return err
	}
	if d.State != StateApproved {
		return d.err("INVALID_STATE", "只有已批准档案可冻结")
	}
	p, ok := d.CurrentPlan()
	if !ok {
		return d.err("PLAN_MISSING", "缺少方案")
	}
	if p.Decision != DecisionApprove {
		return d.err("PLAN_NOT_APPROVED", "当前方案尚未批准")
	}
	for _, f := range d.Findings {
		if f.Blocking && f.Status != FindingResolved {
			return d.err("BLOCKING_FINDING", "冻结前仍存在未解决阻断问题")
		}
	}
	ids := []string{}
	for id := range d.Components {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	obs := []string{}
	for _, id := range ids {
		o, ok := d.LatestObservation(id)
		if !ok {
			return d.err("MISSING_OBSERVATION", "冻结清单缺少构件最新观察")
		}
		obs = append(obs, o.ID)
		if SeverityRank(o.Severity) >= SeverityRank(SeverityHigh) && !d.PlanCoversComponent(*p, id) {
			return d.err("HIGH_RISK_UNCOVERED", "当前方案未完整覆盖高风险构件")
		}
	}
	m := &FreezeManifest{DossierID: d.ID, DossierVersion: d.Version, ComponentIDs: ids, ObservationIDs: obs, PlanID: p.ID, PlanRevision: p.Revision, FrozenAt: time.Now().UTC()}
	m.ManifestHash = d.manifestDigest(m, *p)
	if expectedHash != "" && expectedHash != m.ManifestHash {
		return d.err("MANIFEST_MISMATCH", "提交的清单摘要与档案重算值不一致")
	}
	d.Manifest = m
	if err := d.Transition(StateFrozen); err != nil {
		return err
	}
	d.bump()
	return nil
}
func (d *SurveyDossier) Release(actor, audit string) (WorkReleaseCertificate, error) {
	if d.Certificate != nil {
		return *d.Certificate, nil
	}
	if d.State != StateFrozen {
		return WorkReleaseCertificate{}, d.err("INVALID_STATE", "只有冻结档案可放行")
	}
	if !d.VerifyManifest() {
		return WorkReleaseCertificate{}, d.err("MANIFEST_INVALID", "冻结清单校验失败")
	}
	c := WorkReleaseCertificate{ID: Digest(struct {
		D string
		N int64
	}{d.ID, time.Now().UnixNano()})[:20], DossierID: d.ID, DossierVersion: d.Version, FreezeManifestHash: d.Manifest.ManifestHash, AuditHeadHash: audit, IssuedBy: actor, IssuedAt: time.Now().UTC()}
	c.CertificateDigest = c.Digest()
	d.Certificate = &c
	d.Transition(StateReleased)
	d.bump()
	return c, nil
}

// manifestDigest binds the frozen identity list to the content it identifies.
// This lets release verification detect edits to a component, observation, or
// approved plan even when their IDs and revisions are unchanged.
func (d *SurveyDossier) manifestDigest(m *FreezeManifest, plan RepairPlanRevision) string {
	components := make(map[string]TimberComponent, len(m.ComponentIDs))
	for _, id := range m.ComponentIDs {
		component, ok := d.Components[id]
		if !ok {
			return ""
		}
		components[id] = component
	}
	observations := make([]ConditionObservation, 0, len(m.ObservationIDs))
	for _, wanted := range m.ObservationIDs {
		found := false
		for _, history := range d.Observations {
			for _, observation := range history {
				if observation.ID == wanted {
					observations = append(observations, observation)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return ""
		}
	}
	sort.Slice(observations, func(i, j int) bool { return observations[i].ID < observations[j].ID })
	return Digest(struct {
		DossierID      string
		DossierVersion uint64
		ComponentIDs   []string
		ObservationIDs []string
		PlanID         string
		PlanRevision   uint32
		Components     map[string]TimberComponent
		Observations   []ConditionObservation
		Plan           RepairPlanRevision
	}{m.DossierID, m.DossierVersion, m.ComponentIDs, m.ObservationIDs, m.PlanID, m.PlanRevision, components, observations, plan})
}
