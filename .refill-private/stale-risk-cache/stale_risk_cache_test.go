package staleriskcache_test

import (
	"testing"
	"time"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/domain"
	"timber-release-gate/internal/store"
)

func TestRiskCacheRefreshesAfterCommittedPlan(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := application.New(st)
	dossier, err := svc.Create(application.CreateInput{
		BuildingCode:   "CACHE-BUILDING",
		Title:          "风险缓存版本隔离复现",
		SurveyBoundary: "一层承重柱",
	}, "cache-create", "surveyor")
	if err != nil {
		t.Fatal(err)
	}
	dossier, err = svc.AddComponent(dossier.ID, application.ComponentInput{
		ComponentCode:  "CACHE-COLUMN",
		ComponentType:  "柱",
		RequiredChecks: []string{"病害"},
	}, dossier.Version, "surveyor")
	if err != nil {
		t.Fatal(err)
	}
	componentID := onlyComponentID(t, dossier)
	dossier, err = svc.Observe(dossier.ID, application.ObservationInput{
		ComponentID:    componentID,
		ConditionType:  "柱脚腐朽",
		LocationDetail: "柱脚北侧",
		Severity:       domain.SeverityHigh,
		Measurements:   map[string]float64{"decayDepth": 12},
		EvidenceRefs:   []string{"cache-photo-1"},
		ObservedAt:     time.Now().UTC().Add(-time.Minute),
	}, dossier.Version, "surveyor", "cache-observe")
	if err != nil {
		t.Fatal(err)
	}
	result, dossier, err := svc.Assess(dossier.ID, dossier.Version, "assessor")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("预期一个高风险问题项，实际为%d", len(result.Findings))
	}
	before, err := svc.QueryRisk(dossier.ID, application.RiskFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Items) != 1 || before.Items[0].Covered || before.Summary.CurrentPlanRevision != 0 {
		t.Fatalf("首次风险查询不符合复现前提: %#v", before)
	}
	observationID := dossier.Observations[componentID][0].ID
	dossier, err = svc.SubmitPlan(dossier.ID, application.PlanInput{
		Revision:                 1,
		ReferencedObservationIDs: []string{observationID},
		Actions: []domain.RepairAction{{
			FindingID:          result.Findings[0].ID,
			ComponentID:        componentID,
			Action:             "更换腐朽段并加固",
			MaterialConstraint: "同树种干燥木材",
			AcceptanceStandard: "腐朽段清除且连接牢固",
		}},
	}, dossier.Version, "planner")
	if err != nil {
		t.Fatal(err)
	}
	if dossier.State != domain.StatePlanSubmitted {
		t.Fatalf("方案未提交: state=%s", dossier.State)
	}
	after, err := svc.QueryRisk(dossier.ID, application.RiskFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Items) != 1 || !after.Items[0].Covered || after.Summary.CurrentPlanRevision != 1 || after.Summary.CoveredCount != 1 {
		t.Fatalf("方案提交后风险缓存仍返回旧版本: %#v", after)
	}
}

func onlyComponentID(t *testing.T, dossier *domain.SurveyDossier) string {
	t.Helper()
	for id := range dossier.Components {
		return id
	}
	t.Fatal("档案缺少构件")
	return ""
}
