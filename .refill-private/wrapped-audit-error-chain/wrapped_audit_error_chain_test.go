package wrappedauditerrorchain_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/httpapi"
	"timber-release-gate/internal/store"
)

func TestWrappedAuditErrorPreservesStableHTTPCode(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := application.New(st)
	dossier, err := svc.Create(application.CreateInput{
		BuildingCode:   "LAB-ERROR-CHAIN",
		Title:          "审计错误链复现",
		SurveyBoundary: "校准区",
	}, "audit-error-create", "tester")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(st.DataPath("events.jsonl"), []byte("{broken\n"), 0644); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dossiers/"+dossier.ID+"/timeline", nil)
	response := httptest.NewRecorder()
	httpapi.New(svc).Handler().ServeHTTP(response, request)

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusConflict || body.Error.Code != "AUDIT_INTEGRITY_ERROR" || !strings.Contains(body.Error.Message, "审计链验证失败") {
		t.Fatalf("包装后的审计错误丢失稳定语义或上下文: status=%d code=%s message=%q", response.Code, body.Error.Code, body.Error.Message)
	}
}
