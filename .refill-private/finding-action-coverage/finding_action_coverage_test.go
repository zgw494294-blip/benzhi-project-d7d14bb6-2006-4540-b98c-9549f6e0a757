package findingactioncoverage_test

import (
	"testing"
	"time"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/domain"
	"timber-release-gate/internal/store"
)

func TestFindingOnlyActionCoversComponentAcrossRiskAndFreeze(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := application.New(st)
	dossier, err := svc.Create(application.CreateInput{
		BuildingCode: "FINDING-COVERAGE", Title: "问题项覆盖检查", SurveyBoundary: "正殿",
	}, "finding-create", "tester")
	if err != nil {
		t.Fatal(err)
	}
	dossier, err = svc.AddComponent(dossier.ID, application.ComponentInput{
		ComponentCode: "C-01", ComponentType: "柱", RequiredChecks: []string{"开裂"},
	}, dossier.Version, "tester")
	if err != nil {
		t.Fatal(err)
	}
	componentID := domain.Digest(struct{ D, C string }{dossier.ID, "C-01"})[:20]
	dossier, err = svc.Observe(dossier.ID, application.ObservationInput{
		ComponentID: componentID, ConditionType: "开裂", LocationDetail: "柱脚",
		Severity: domain.SeverityHigh, Measurements: map[string]float64{"crackWidth": 2},
		EvidenceRefs: []string{"photo-1"}, ObservedAt: time.Now().UTC().Add(-time.Minute),
	}, dossier.Version, "tester", "finding-observe")
	if err != nil {
		t.Fatal(err)
	}
	result, dossier, err := svc.Assess(dossier.ID, dossier.Version, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("预期一个高风险问题项，实际为 %#v", result.Findings)
	}
	findingID := result.Findings[0].ID
	observationID := dossier.Observations[componentID][0].ID
	dossier, err = svc.SubmitPlan(dossier.ID, application.PlanInput{
		Revision:                 1,
		ReferencedObservationIDs: []string{observationID},
		Actions: []domain.RepairAction{{
			FindingID: findingID, Action: "墩接加固",
			MaterialConstraint: "同树种干燥木材", AcceptanceStandard: "连接牢固",
		}},
	}, dossier.Version, "tester")
	if err != nil {
		t.Fatalf("问题项动作未通过方案校验: %v", err)
	}
	risk, err := svc.QueryRisk(dossier.ID, application.RiskFilter{})
	if err != nil {
		t.Fatal(err)
	}
	summaryCovered := len(risk.Items) == 1 && risk.Items[0].Covered && risk.Summary.CoveredCount == 1

	dossier, err = svc.Review(dossier.ID, domain.DecisionApprove, []string{"覆盖完整"}, []string{findingID}, dossier.Version, "tester")
	if err != nil {
		t.Fatal(err)
	}
	_, freezeErr := svc.Freeze(dossier.ID, dossier.Version, "tester")
	if !summaryCovered || freezeErr != nil {
		t.Fatalf("问题项动作的组件覆盖未跨层保持一致: risk=%#v freezeErr=%v", risk, freezeErr)
	}
}
