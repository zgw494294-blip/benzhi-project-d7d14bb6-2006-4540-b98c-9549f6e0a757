package domain

func (d *SurveyDossier) CanRelease() error {
	if d.State != StateFrozen {
		return d.err("INVALID_STATE", "档案尚未冻结")
	}
	if d.Manifest == nil || d.Manifest.ManifestHash == "" {
		return d.err("MANIFEST_MISSING", "冻结清单摘要缺失")
	}
	if d.Certificate != nil && d.Certificate.CertificateDigest == "" {
		return d.err("CERTIFICATE_INVALID", "已有凭据摘要缺失")
	}
	return nil
}
func (d *SurveyDossier) VerifyManifest() bool {
	if d.Manifest == nil {
		return false
	}
	p, ok := d.CurrentPlan()
	if !ok {
		return false
	}
	if d.Manifest.PlanID != p.ID || d.Manifest.PlanRevision != p.Revision {
		return false
	}
	return d.Manifest.ManifestHash == d.manifestDigest(d.Manifest, *p)
}
