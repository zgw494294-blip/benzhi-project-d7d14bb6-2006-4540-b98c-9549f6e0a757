package truncatedjsontail_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/httpapi"
	"timber-release-gate/internal/store"
)

func TestOversizedTrailingJSONIsRejectedWithoutCommit(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := httpapi.New(application.New(st)).Handler()
	valid := []byte(`{"buildingCode":"LIMIT","title":"请求边界","surveyBoundary":"正殿"}`)
	body := append(valid, bytes.Repeat([]byte{' '}, (1<<20)-len(valid))...)
	body = append(body, []byte("  {\"unexpected\":true}")...)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/dossiers", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "oversized-tail")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || st.Count() != 0 {
		t.Fatalf("超限尾随JSON被接受并进入写入链: status=%d count=%d body=%s", response.Code, st.Count(), strings.TrimSpace(response.Body.String()))
	}
}
