package sharedledgerwriters_test

import (
	"testing"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/store"
)

func TestSharedDirectoryWritersPreserveLedgerChain(t *testing.T) {
	dir := t.TempDir()
	firstStore, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	secondStore, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	first, err := application.New(firstStore).Create(application.CreateInput{
		BuildingCode: "SHARED-1", Title: "一号勘察", SurveyBoundary: "正殿",
	}, "shared-create-1", "tester")
	if err != nil {
		t.Fatalf("第一个写入失败: %v", err)
	}
	second, err := application.New(secondStore).Create(application.CreateInput{
		BuildingCode: "SHARED-2", Title: "二号勘察", SurveyBoundary: "偏殿",
	}, "shared-create-2", "tester")
	if err != nil {
		t.Fatalf("第二个写入失败: %v", err)
	}

	reopened, err := store.Open(dir)
	if err != nil {
		t.Fatalf("两个成功写入破坏了可重开账本: %v", err)
	}
	if _, ok := reopened.Get(first.ID); !ok {
		t.Fatalf("重开后缺少第一个档案 %s", first.ID)
	}
	if _, ok := reopened.Get(second.ID); !ok {
		t.Fatalf("重开后缺少第二个档案 %s", second.ID)
	}
	if summary := reopened.Summary(); summary.Count != 2 || summary.LastSequence != 2 {
		t.Fatalf("重开后的账本摘要不连续: %#v", summary)
	}
}
