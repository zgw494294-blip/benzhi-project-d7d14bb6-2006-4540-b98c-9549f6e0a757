package application

import (
	"sync"
	"timber-release-gate/internal/assessment"
	"timber-release-gate/internal/domain"
	"timber-release-gate/internal/store"
	"time"
)

type Service struct {
	st        *store.Store
	mu        sync.Mutex
	riskMu    sync.Mutex
	riskCache map[string]riskCacheEntry
}

func New(st *store.Store) *Service {
	return &Service{st: st, riskCache: map[string]riskCacheEntry{}}
}

type CreateInput struct{ BuildingCode, Title, SurveyBoundary string }
type ComponentInput struct {
	ComponentCode, ComponentType, Location, LoadPathParentCode, BaselineNote string
	RequiredChecks                                                           []string
}

func (s *Service) Create(in CreateInput, key, actor string) (*domain.SurveyDossier, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, e := domain.NewDossier(in.BuildingCode, in.Title, in.SurveyBoundary)
	if e != nil {
		return nil, e
	}
	h := domain.Digest(in)
	if key != "" {
		if old, ok, er := s.st.Idempotent(key, h); ok || er != nil {
			return old, er
		}
	}
	e = s.st.Commit(d, 0, key, h, "DOSSIER_CREATED", actor)
	if e == nil {
		s.invalidateRisk(d.ID)
	}
	return d, e
}
func (s *Service) mutate(id string, expected uint64, actor, typ string, fn func(*domain.SurveyDossier) error) (*domain.SurveyDossier, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.st.Get(id)
	if !ok {
		return nil, &domain.Error{Code: "NOT_FOUND", Message: "档案不存在"}
	}
	if expected == 0 {
		return nil, &domain.Error{Code: "EXPECTED_VERSION_REQUIRED", Message: "写入必须提供expectedVersion", State: d.State, Version: d.Version}
	}
	if d.Version != expected {
		return nil, &domain.Error{Code: "VERSION_CONFLICT", Message: "档案版本已变化", State: d.State, Version: d.Version}
	}
	if e := fn(d); e != nil {
		return nil, e
	}
	if e := s.st.Commit(d, expected, "", domain.Digest(d), typ, actor); e != nil {
		return nil, e
	}
	s.invalidateRisk(d.ID)
	return d, nil
}
func (s *Service) AddComponent(id string, in ComponentInput, expected uint64, actor string) (*domain.SurveyDossier, error) {
	return s.AddComponents(id, []ComponentInput{in}, expected, actor, "")
}

func (s *Service) AddComponents(id string, inputs []ComponentInput, expected uint64, actor, key string) (*domain.SurveyDossier, error) {
	return s.mutateKey(id, inputs, expected, actor, key, "COMPONENT_ADDED", func(d *domain.SurveyDossier) error {
		components := make([]domain.TimberComponent, 0, len(inputs))
		for _, in := range inputs {
			components = append(components, domain.TimberComponent{ComponentCode: in.ComponentCode, ComponentType: in.ComponentType, Location: in.Location, LoadPathParentCode: in.LoadPathParentCode, BaselineNote: in.BaselineNote, RequiredChecks: in.RequiredChecks})
		}
		return d.AddComponents(components)
	})
}

type ObservationInput struct {
	ComponentID, ConditionType, LocationDetail string
	Severity                                   domain.Severity
	Measurements                               map[string]float64
	EvidenceRefs                               []string
	ObservedAt                                 time.Time
}

func (s *Service) Observe(id string, in ObservationInput, expected uint64, actor, key string) (*domain.SurveyDossier, error) {
	return s.mutateKey(id, in, expected, actor, key, "OBSERVATION_ADDED", func(d *domain.SurveyDossier) error {
		return d.AddObservation(domain.ConditionObservation{ComponentID: in.ComponentID, ConditionType: in.ConditionType, LocationDetail: in.LocationDetail, Severity: in.Severity, Measurements: in.Measurements, EvidenceRefs: in.EvidenceRefs, ObservedAt: in.ObservedAt})
	})
}
func (s *Service) mutateKey(id string, in any, expected uint64, actor, key, typ string, fn func(*domain.SurveyDossier) error) (*domain.SurveyDossier, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := domain.Digest(struct {
		DossierID, Type string
		Input           any
	}{id, typ, in})
	if key != "" {
		if old, ok, e := s.st.Idempotent(key, h); ok || e != nil {
			return old, e
		}
	}
	d, ok := s.st.Get(id)
	if !ok {
		return nil, &domain.Error{Code: "NOT_FOUND", Message: "档案不存在"}
	}
	if expected == 0 {
		return nil, &domain.Error{Code: "EXPECTED_VERSION_REQUIRED", Message: "写入必须提供expectedVersion", State: d.State, Version: d.Version}
	}
	if d.Version != expected {
		return nil, &domain.Error{Code: "VERSION_CONFLICT", Message: "档案版本已变化", State: d.State, Version: d.Version}
	}
	if e := fn(d); e != nil {
		return nil, e
	}
	if e := s.st.Commit(d, expected, key, h, typ, actor); e != nil {
		return nil, e
	}
	s.invalidateRisk(d.ID)
	return d, nil
}
func (s *Service) Assess(id string, expected uint64, actor string) (assessment.Result, *domain.SurveyDossier, error) {
	return s.AssessKey(id, expected, actor, "")
}

func (s *Service) AssessKey(id string, expected uint64, actor, key string) (assessment.Result, *domain.SurveyDossier, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hash := domain.Digest(struct{ DossierID, Type string }{id, "ASSESSED"})
	if key != "" {
		if old, ok, err := s.st.Idempotent(key, hash); ok || err != nil {
			if err != nil {
				return assessment.Result{}, old, err
			}
			return assessment.Run(old), old, nil
		}
	}
	d, ok := s.st.Get(id)
	if !ok {
		return assessment.Result{}, nil, &domain.Error{Code: "NOT_FOUND", Message: "档案不存在"}
	}
	if expected == 0 {
		return assessment.Result{}, d, &domain.Error{Code: "EXPECTED_VERSION_REQUIRED", Message: "写入必须提供expectedVersion", State: d.State, Version: d.Version}
	}
	if d.Version != expected {
		return assessment.Result{}, d, &domain.Error{Code: "VERSION_CONFLICT", Message: "档案版本已变化", State: d.State, Version: d.Version}
	}
	if e := d.Mutable(); e != nil {
		return assessment.Result{}, d, e
	}
	if d.State != domain.StateSurveying && d.State != domain.StateAssessed {
		return assessment.Result{}, d, &domain.Error{Code: "INVALID_STATE", Message: "只有勘察中或已校核档案可执行校核", State: d.State, Version: d.Version}
	}
	r := assessment.Apply(d)
	if e := s.st.Commit(d, expected, key, hash, "ASSESSED", actor); e != nil {
		return r, nil, e
	}
	s.invalidateRisk(d.ID)
	return r, d, nil
}

type PlanInput struct {
	Revision                                                                              uint32
	Actions                                                                               []domain.RepairAction
	ReferencedObservationIDs, ResolvedFindingIDs, MaterialConstraints, AcceptanceCriteria []string
}

func (s *Service) SubmitPlan(id string, in PlanInput, expected uint64, actor string) (*domain.SurveyDossier, error) {
	return s.SubmitPlanKey(id, in, expected, actor, "")
}
func (s *Service) SubmitPlanKey(id string, in PlanInput, expected uint64, actor, key string) (*domain.SurveyDossier, error) {
	return s.mutateKey(id, in, expected, actor, key, "PLAN_SUBMITTED", func(d *domain.SurveyDossier) error {
		return d.SubmitPlan(domain.RepairPlanRevision{Revision: in.Revision, Actions: in.Actions, ReferencedObservationIDs: in.ReferencedObservationIDs, ResolvedFindingIDs: in.ResolvedFindingIDs, MaterialConstraints: in.MaterialConstraints, AcceptanceCriteria: in.AcceptanceCriteria})
	})
}
func (s *Service) Review(id string, decision domain.ReviewDecision, comments, resolved []string, expected uint64, actor string) (*domain.SurveyDossier, error) {
	return s.ReviewKey(id, decision, comments, resolved, expected, actor, "")
}
func (s *Service) ReviewKey(id string, decision domain.ReviewDecision, comments, resolved []string, expected uint64, actor, key string) (*domain.SurveyDossier, error) {
	input := struct {
		Decision domain.ReviewDecision
		Comments []string
		Resolved []string
	}{decision, comments, resolved}
	return s.mutateKey(id, input, expected, actor, key, "REVIEW_RECORDED", func(d *domain.SurveyDossier) error { return d.Review(decision, comments, resolved) })
}
func (s *Service) Freeze(id string, expected uint64, actor string) (*domain.SurveyDossier, error) {
	return s.FreezeWithHash(id, "", expected, actor)
}
func (s *Service) FreezeWithHash(id, manifestHash string, expected uint64, actor string) (*domain.SurveyDossier, error) {
	return s.FreezeWithHashKey(id, manifestHash, expected, actor, "")
}
func (s *Service) FreezeWithHashKey(id, manifestHash string, expected uint64, actor, key string) (*domain.SurveyDossier, error) {
	return s.mutateKey(id, manifestHash, expected, actor, key, "DOSSIER_FROZEN", func(d *domain.SurveyDossier) error { return d.FreezeWithHash(manifestHash) })
}
func (s *Service) Release(id, actor string, expected uint64) (*domain.SurveyDossier, error) {
	return s.ReleaseKey(id, actor, expected, "")
}
func (s *Service) ReleaseKey(id, actor string, expected uint64, key string) (*domain.SurveyDossier, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.st.VerifyChain(); err != nil {
		return nil, &domain.Error{Code: "AUDIT_INTEGRITY_ERROR", Message: err.Error()}
	}
	hash := domain.Digest(struct{ DossierID, Type string }{id, "RELEASE_ISSUED"})
	if key != "" {
		if old, ok, err := s.st.Idempotent(key, hash); ok || err != nil {
			if err != nil {
				return nil, err
			}
			if err := validateReleaseCertificate(old, s.st.Events(id)); err != nil {
				return nil, err
			}
			return old, nil
		}
	}
	d, ok := s.st.Get(id)
	if !ok {
		return nil, &domain.Error{Code: "NOT_FOUND", Message: "档案不存在"}
	}
	if d.Certificate != nil {
		if err := validateReleaseCertificate(d, s.st.Events(id)); err != nil {
			return nil, err
		}
		return d, nil
	}
	if expected == 0 {
		return nil, &domain.Error{Code: "EXPECTED_VERSION_REQUIRED", Message: "写入必须提供expectedVersion", State: d.State, Version: d.Version}
	}
	if d.Version != expected {
		return nil, &domain.Error{Code: "VERSION_CONFLICT", Message: "档案版本已变化", State: d.State, Version: d.Version}
	}
	if _, err := d.Release(actor, s.st.HeadHash()); err != nil {
		return nil, err
	}
	if err := s.st.Commit(d, expected, key, hash, "RELEASE_ISSUED", actor); err != nil {
		return nil, err
	}
	s.invalidateRisk(d.ID)
	return d, nil
}
func (s *Service) Timeline(id string) (domain.Timeline, error) {
	if err := s.st.VerifyChain(); err != nil {
		return domain.Timeline{}, &domain.Error{Code: "AUDIT_INTEGRITY_ERROR", Message: err.Error()}
	}
	return s.st.Timeline(id)
}
func (s *Service) Risk(id string) ([]map[string]any, error) {
	response, err := s.QueryRisk(id, RiskFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(response.Items))
	for _, item := range response.Items {
		out = append(out, map[string]any{"findingID": item.FindingID, "code": item.Code, "componentID": item.ComponentID, "componentCode": item.ComponentCode, "severity": item.Severity, "blocking": item.Blocking, "status": item.Status, "covered": item.Covered})
	}
	return out, nil
}
func (s *Service) Certificate(id string) (*domain.WorkReleaseCertificate, error) {
	if err := s.st.VerifyChain(); err != nil {
		return nil, &domain.Error{Code: "AUDIT_INTEGRITY_ERROR", Message: err.Error()}
	}
	d, ok := s.st.Get(id)
	if !ok || d.Certificate == nil {
		return nil, &domain.Error{Code: "NOT_FOUND", Message: "放行凭据不存在"}
	}
	if err := validateReleaseCertificate(d, s.st.Events(id)); err != nil {
		return nil, err
	}
	c := *d.Certificate
	return &c, nil
}

func validateReleaseCertificate(d *domain.SurveyDossier, events []domain.AuditEvent) error {
	if d.State != domain.StateReleased || d.Certificate == nil {
		return &domain.Error{Code: "NOT_RELEASED", Message: "档案尚未完成施工放行"}
	}
	if !d.VerifyManifest() {
		return &domain.Error{Code: "MANIFEST_INTEGRITY_ERROR", Message: "冻结清单校验失败"}
	}
	c := d.Certificate
	if c.Digest() != c.CertificateDigest || c.FreezeManifestHash != d.Manifest.ManifestHash {
		return &domain.Error{Code: "CERTIFICATE_INTEGRITY_ERROR", Message: "凭据摘要校验失败"}
	}
	for _, event := range events {
		if event.EventType == "RELEASE_ISSUED" {
			if event.PreviousHash != c.AuditHeadHash {
				return &domain.Error{Code: "CERTIFICATE_INTEGRITY_ERROR", Message: "凭据审计头哈希不匹配"}
			}
			return nil
		}
	}
	return &domain.Error{Code: "CERTIFICATE_INTEGRITY_ERROR", Message: "缺少放行审计事件"}
}
