package store

import (
	"os"
	"path/filepath"
	"testing"

	"timber-release-gate/internal/domain"
)

func TestOpenRebuildsCorruptedSnapshotFromLedger(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	dossier, err := domain.NewDossier("B-RECOVERY", "恢复测试", "正殿")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Commit(dossier, 0, "recovery-create", domain.Digest("create"), "DOSSIER_CREATED", "tester"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "snapshot.json"), []byte("{broken"), 0644); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	restored, ok := reopened.Get(dossier.ID)
	if !ok || restored.Version != dossier.Version || reopened.Summary().Count != 1 {
		t.Fatalf("账本重放结果不正确: restored=%#v", restored)
	}
}

func TestOpenRejectsPartialLedgerTail(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	dossier, _ := domain.NewDossier("B-TAIL", "尾部测试", "正殿")
	if err := st.Commit(dossier, 0, "tail-create", domain.Digest("create"), "DOSSIER_CREATED", "tester"); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("{partial"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("尾部残缺账本未被拒绝")
	}
}
