package failedsnapshotcommit_test

import (
	"os"
	"path/filepath"
	"testing"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/store"
)

func TestFailedSnapshotCommitDoesNotPublishDossier(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	snapshotPath := filepath.Join(dir, "snapshot.json")
	if err := os.Mkdir(snapshotPath, 0755); err != nil {
		t.Fatal(err)
	}

	dossier, commitErr := application.New(st).Create(application.CreateInput{
		BuildingCode: "SNAPSHOT-FAIL", Title: "快照失败检查", SurveyBoundary: "正殿",
	}, "snapshot-failure", "tester")
	if commitErr == nil {
		t.Fatal("快照替换失效时创建调用意外成功")
	}
	_, visibleInProcess := st.Get(dossier.ID)

	if err := os.Remove(snapshotPath); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dir)
	if err != nil {
		t.Fatalf("清除失效目标后账本无法重开: %v", err)
	}
	_, visibleAfterRestart := reopened.Get(dossier.ID)
	if visibleInProcess || visibleAfterRestart {
		t.Fatalf("返回失败的创建被发布: inProcess=%t afterRestart=%t", visibleInProcess, visibleAfterRestart)
	}
}
