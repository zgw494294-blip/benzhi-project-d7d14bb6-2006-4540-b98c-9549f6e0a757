package application

import (
	"testing"
	"time"

	"timber-release-gate/internal/domain"
	"timber-release-gate/internal/store"
)

func TestBatchComponentsAreAtomicAndIdempotent(t *testing.T) {
	svc := testService(t)
	d, err := svc.Create(CreateInput{BuildingCode: "B-01", Title: "正殿", SurveyBoundary: "正殿"}, "create-1", "surveyor")
	if err != nil {
		t.Fatal(err)
	}
	d, err = svc.AddComponent(d.ID, ComponentInput{ComponentCode: "C-01", ComponentType: "柱"}, d.Version, "surveyor")
	if err != nil {
		t.Fatal(err)
	}
	version, eventCount := d.Version, len(svc.st.Events(d.ID))
	_, err = svc.AddComponents(d.ID, []ComponentInput{{ComponentCode: "C-02", ComponentType: "梁"}, {ComponentCode: "C-01", ComponentType: "柱"}}, version, "surveyor", "failed-batch")
	assertDomainError(t, err, "DUPLICATE_COMPONENT")
	unchanged, _ := svc.st.Get(d.ID)
	if unchanged.Version != version || len(unchanged.Components) != 1 || len(svc.st.Events(d.ID)) != eventCount {
		t.Fatalf("失败批次改变了档案: version=%d components=%d events=%d", unchanged.Version, len(unchanged.Components), len(svc.st.Events(d.ID)))
	}
	batch := []ComponentInput{{ComponentCode: "C-02", ComponentType: "梁", LoadPathParentCode: "C-01"}, {ComponentCode: "C-03", ComponentType: "檩", LoadPathParentCode: "C-02"}}
	first, err := svc.AddComponents(d.ID, batch, version, "surveyor", "batch-1")
	if err != nil {
		t.Fatal(err)
	}
	retry, err := svc.AddComponents(d.ID, batch, version, "surveyor", "batch-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != version+1 || retry.Version != first.Version || len(first.Components) != 3 || len(svc.st.Events(d.ID)) != eventCount+1 {
		t.Fatalf("批量幂等结果不正确: first=%d retry=%d components=%d events=%d", first.Version, retry.Version, len(first.Components), len(svc.st.Events(d.ID)))
	}
}

func TestObservationReassessmentTracksRemovedFinding(t *testing.T) {
	svc, d, componentID := dossierWithComponent(t)
	observedAt := time.Now().UTC().Add(-time.Minute)
	d, err := svc.Observe(d.ID, ObservationInput{ComponentID: componentID, ConditionType: "腐朽", LocationDetail: "柱脚", Severity: domain.SeverityHigh, Measurements: map[string]float64{"decayDepth": 12}, EvidenceRefs: []string{" photo-1 ", "photo-1"}, ObservedAt: observedAt}, d.Version, "surveyor", "observe-1")
	if err != nil {
		t.Fatal(err)
	}
	first := d.Observations[componentID][0]
	if len(first.EvidenceRefs) != 1 || first.EvidenceRefs[0] != "photo-1" {
		t.Fatalf("证据未规范化: %#v", first.EvidenceRefs)
	}
	result, d, err := svc.Assess(d.ID, d.Version, "assessor")
	if err != nil {
		t.Fatal(err)
	}
	if d.State != domain.StateAssessed || len(result.Findings) != 1 || result.Findings[0].Code != "HIGH_RISK" {
		t.Fatalf("首次校核结果不正确: state=%s findings=%#v", d.State, result.Findings)
	}
	highRiskID := result.Findings[0].ID
	d, err = svc.Observe(d.ID, ObservationInput{ComponentID: componentID, ConditionType: "腐朽复测", LocationDetail: "柱脚", Severity: domain.SeverityLow, Measurements: map[string]float64{"decayDepth": 2}, EvidenceRefs: []string{"photo-2"}, ObservedAt: observedAt.Add(time.Second)}, d.Version, "surveyor", "observe-2")
	if err != nil {
		t.Fatal(err)
	}
	second := d.Observations[componentID][1]
	if second.Revision != 2 || second.SupersedesID != first.ID {
		t.Fatalf("观察修订链不正确: %#v", second)
	}
	result, d, err = svc.Assess(d.ID, d.Version, "assessor")
	if err != nil {
		t.Fatal(err)
	}
	foundRemoved := false
	for _, change := range result.Changes {
		foundRemoved = foundRemoved || change.Change == domain.FindingRemoved && change.Finding.ID == highRiskID
	}
	if !foundRemoved || len(d.Findings) != 0 {
		t.Fatalf("重跑未标记消失问题: changes=%#v findings=%#v", result.Changes, d.Findings)
	}
	view, err := svc.View(d.ID)
	if err != nil {
		t.Fatal(err)
	}
	history := view["observationHistory"].([]ObservationHistoryItem)
	if len(history) != 2 || history[0].Current || !history[1].Current {
		t.Fatalf("观察历史当前标识不正确: %#v", history)
	}
}

func TestHighRiskPlanFreezeAndRelease(t *testing.T) {
	svc, d, componentID := dossierWithComponent(t)
	_, err := svc.Observe(d.ID, ObservationInput{ComponentID: componentID, ConditionType: "开裂", LocationDetail: "柱身", Severity: domain.SeverityCritical, Measurements: map[string]float64{"crackWidth": 8}, EvidenceRefs: []string{"photo-critical"}, ObservedAt: time.Now().UTC().Add(-time.Minute)}, d.Version, "surveyor", "critical-observation")
	if err != nil {
		t.Fatal(err)
	}
	d, _ = svc.st.Get(d.ID)
	result, d, err := svc.Assess(d.ID, d.Version, "assessor")
	if err != nil {
		t.Fatal(err)
	}
	findingID := result.Findings[0].ID
	observationID := d.Observations[componentID][0].ID
	plan := PlanInput{Revision: 1, ReferencedObservationIDs: []string{observationID}, Actions: []domain.RepairAction{{ComponentID: componentID, Action: "墩接加固", MaterialConstraint: "同树种干燥木材", AcceptanceStandard: "连接牢固且含水率合格"}}}
	d, err = svc.SubmitPlan(d.ID, plan, d.Version, "planner")
	if err != nil {
		t.Fatal(err)
	}
	risk, err := svc.QueryRisk(d.ID, RiskFilter{})
	if err != nil || len(risk.Items) != 1 || !risk.Items[0].Covered || risk.Summary.HighRiskComponents != 1 {
		t.Fatalf("风险覆盖统计不正确: risk=%#v err=%v", risk, err)
	}
	d, err = svc.Review(d.ID, domain.DecisionApprove, []string{"风险动作和验收条件完整"}, []string{findingID}, d.Version, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	d, err = svc.Freeze(d.ID, d.Version, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if d.Manifest == nil || len(d.Manifest.ObservationIDs) != 1 || d.Manifest.ObservationIDs[0] != observationID || !d.VerifyManifest() {
		t.Fatalf("冻结清单不正确: %#v", d.Manifest)
	}
	component := d.Components[componentID]
	originalNote := component.BaselineNote
	component.BaselineNote = "被篡改的基线"
	d.Components[componentID] = component
	if d.VerifyManifest() {
		t.Fatal("冻结清单摘要未绑定构件内容")
	}
	component.BaselineNote = originalNote
	d.Components[componentID] = component
	frozenVersion := d.Version
	first, err := svc.Release(d.ID, "issuer", frozenVersion)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := svc.Release(d.ID, "issuer", frozenVersion)
	if err != nil {
		t.Fatal(err)
	}
	if first.Certificate.ID != retry.Certificate.ID || len(svc.st.EventsByType(d.ID, "RELEASE_ISSUED")) != 1 {
		t.Fatalf("重复放行产生了新凭据或事件")
	}
	if _, err := svc.Certificate(d.ID); err != nil {
		t.Fatal(err)
	}
}

func TestReturnedPlanCreatesBlockingFindingAndRequiresHigherRevision(t *testing.T) {
	svc, d, componentID := dossierWithComponent(t)
	d, err := svc.Observe(d.ID, ObservationInput{
		ComponentID: componentID, ConditionType: "轻微开裂", LocationDetail: "柱身",
		Severity: domain.SeverityLow, Measurements: map[string]float64{"crackWidth": 0.2},
		EvidenceRefs: []string{"photo-low"}, ObservedAt: time.Now().UTC().Add(-time.Minute),
	}, d.Version, "surveyor", "observe-return")
	if err != nil {
		t.Fatal(err)
	}
	_, d, err = svc.Assess(d.ID, d.Version, "assessor")
	if err != nil {
		t.Fatal(err)
	}
	observationID := d.Observations[componentID][0].ID
	basePlan := PlanInput{Revision: 1, ReferencedObservationIDs: []string{observationID}, Actions: []domain.RepairAction{{ComponentID: componentID, Action: "监测并局部修补", MaterialConstraint: "相容木材", AcceptanceStandard: "裂缝稳定"}}}
	d, err = svc.SubmitPlanKey(d.ID, basePlan, d.Version, "planner", "plan-return-1")
	if err != nil {
		t.Fatal(err)
	}
	eventsBeforeRetry := len(svc.st.Events(d.ID))
	retry, err := svc.SubmitPlanKey(d.ID, basePlan, d.Version-1, "planner", "plan-return-1")
	if err != nil || retry.Version != d.Version || len(svc.st.Events(d.ID)) != eventsBeforeRetry {
		t.Fatalf("方案幂等重试不正确: retry=%#v err=%v", retry, err)
	}
	d, err = svc.Review(d.ID, domain.DecisionReturn, []string{"补充节点验收记录"}, nil, d.Version, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	var reviewFindingID string
	for id, finding := range d.Findings {
		if finding.Source == domain.SourceReview && finding.Blocking && finding.Status == domain.FindingOpen {
			reviewFindingID = id
		}
	}
	if reviewFindingID == "" || d.State != domain.StateChangesRequested {
		t.Fatalf("退回未生成阻断意见: state=%s findings=%#v", d.State, d.Findings)
	}
	history := d.StateHistory(svc.AuditEvents(d.ID))
	if len(history) < 4 || history[len(history)-1] != domain.StateChangesRequested {
		t.Fatalf("退回审核未记录正确状态历史: %#v", history)
	}
	withoutResolution := basePlan
	withoutResolution.Revision = 2
	_, err = svc.SubmitPlan(d.ID, withoutResolution, d.Version, "planner")
	assertDomainError(t, err, "FINDING_UNRESOLVED")
	unchanged, _ := svc.st.Get(d.ID)
	withResolution := withoutResolution
	withResolution.ResolvedFindingIDs = []string{reviewFindingID}
	d, err = svc.SubmitPlan(d.ID, withResolution, unchanged.Version, "planner")
	if err != nil {
		t.Fatal(err)
	}
	d, err = svc.Review(d.ID, domain.DecisionApprove, []string{"复核通过"}, nil, d.Version, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if d.State != domain.StateApproved || d.Findings[reviewFindingID].Status != domain.FindingResolved {
		t.Fatalf("审核意见未闭环: state=%s finding=%#v", d.State, d.Findings[reviewFindingID])
	}
}

func testService(t *testing.T) *Service {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return New(st)
}

func dossierWithComponent(t *testing.T) (*Service, *domain.SurveyDossier, string) {
	t.Helper()
	svc := testService(t)
	d, err := svc.Create(CreateInput{BuildingCode: "B-TEST", Title: "测试殿", SurveyBoundary: "测试范围"}, "create", "surveyor")
	if err != nil {
		t.Fatal(err)
	}
	d, err = svc.AddComponent(d.ID, ComponentInput{ComponentCode: "C-01", ComponentType: "柱", RequiredChecks: []string{"病害"}}, d.Version, "surveyor")
	if err != nil {
		t.Fatal(err)
	}
	return svc, d, domain.Digest(struct{ D, C string }{d.ID, "C-01"})[:20]
}

func assertDomainError(t *testing.T, err error, code string) {
	t.Helper()
	domainErr, ok := err.(*domain.Error)
	if !ok || domainErr.Code != code {
		t.Fatalf("错误不匹配: want=%s got=%v", code, err)
	}
}
