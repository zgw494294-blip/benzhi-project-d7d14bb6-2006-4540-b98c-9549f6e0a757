package staleledgerhandle_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/httpapi"
	"timber-release-gate/internal/store"
)

func TestAtomicLedgerReplacementInvalidatesVerificationHandle(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	svc := application.New(st)
	dossier, err := svc.Create(application.CreateInput{
		BuildingCode:   "B-LEDGER",
		Title:          "账本资源替换复现",
		SurveyBoundary: "校验边界",
	}, "replace-ledger-create", "auditor")
	if err != nil {
		t.Fatal(err)
	}
	handler := httpapi.New(svc).Handler()
	timelinePath := "/api/v1/dossiers/" + dossier.ID + "/timeline"
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, timelinePath, nil))
	if first.Code != http.StatusOK {
		t.Fatalf("预热账本校验失败: status=%d body=%s", first.Code, first.Body.String())
	}

	temporary, err := os.CreateTemp(dataDir, "replacement-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := temporary.Write([]byte("{\"corrupted\":true}\n")); err != nil {
		t.Fatal(err)
	}
	if err := temporary.Sync(); err != nil {
		t.Fatal(err)
	}
	replacementPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacementPath, filepath.Join(dataDir, "events.jsonl")); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, timelinePath, bytes.NewReader(nil)))
	if response.Code != http.StatusConflict {
		t.Fatalf("原子替换损坏账本后仍信任失效句柄: want=%d got=%d body=%s", http.StatusConflict, response.Code, response.Body.String())
	}
}
