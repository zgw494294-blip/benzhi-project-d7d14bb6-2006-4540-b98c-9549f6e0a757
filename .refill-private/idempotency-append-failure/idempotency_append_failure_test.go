package idempotencyappendfailure

import (
	"os"
	"testing"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/store"
)

func TestAppendFailureDoesNotPoisonIdempotentRetry(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := application.New(st)
	dossier, err := svc.Create(application.CreateInput{BuildingCode: "B-01", Title: "声学基线", SurveyBoundary: "实验室一区"}, "", "技术员")
	if err != nil {
		t.Fatal(err)
	}

	eventsPath := st.DataPath("events.jsonl")
	originalLedger, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(eventsPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(eventsPath, 0755); err != nil {
		t.Fatal(err)
	}
	first, err := svc.AddComponents(dossier.ID, []application.ComponentInput{{ComponentCode: "C-01", ComponentType: "柱", Location: "东次间"}}, dossier.Version, "技术员", "retry-once")
	if err == nil || first != nil {
		t.Fatalf("账本追加失败时应返回错误且不返回成功档案，dossier=%v err=%v", first, err)
	}
	if err := os.Remove(eventsPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eventsPath, originalLedger, 0644); err != nil {
		t.Fatal(err)
	}

	retried, err := svc.AddComponents(dossier.ID, []application.ComponentInput{{ComponentCode: "C-01", ComponentType: "柱", Location: "东次间"}}, dossier.Version, "技术员", "retry-once")
	if err != nil {
		t.Fatal(err)
	}
	if retried == nil || len(retried.Components) != 1 {
		t.Fatalf("恢复账本后重试应返回已持久化构件，dossier=%v", retried)
	}
	if err := st.VerifyChain(); err != nil {
		t.Fatalf("重试成功后内存投影与恢复的账本应保持一致，校验失败=%v", err)
	}
}
