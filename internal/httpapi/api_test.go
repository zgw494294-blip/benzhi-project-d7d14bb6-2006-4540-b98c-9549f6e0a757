package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/store"
)

func TestWriteProtocolValidation(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := New(application.New(st)).Handler()
	badType := httptest.NewRequest(http.MethodPost, "/api/v1/dossiers", bytes.NewBufferString(`{"buildingCode":"B","title":"殿","surveyBoundary":"正殿"}`))
	badType.Header.Set("Content-Type", "text/plain")
	badResponse := httptest.NewRecorder()
	handler.ServeHTTP(badResponse, badType)
	if badResponse.Code != http.StatusBadRequest || errorCode(t, badResponse) != "INVALID_CONTENT_TYPE" {
		t.Fatalf("Content-Type错误映射不正确: status=%d body=%s", badResponse.Code, badResponse.Body.String())
	}

	create := httptest.NewRequest(http.MethodPost, "/api/v1/dossiers", bytes.NewBufferString(`{"buildingCode":"B","title":"殿","surveyBoundary":"正殿"}`))
	create.Header.Set("Content-Type", "application/json")
	create.Header.Set("Idempotency-Key", "http-create")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("建档失败: %s", created.Body.String())
	}
	var dossier map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &dossier); err != nil {
		t.Fatal(err)
	}
	id, _ := dossier["id"].(string)
	component := httptest.NewRequest(http.MethodPost, "/api/v1/dossiers/"+id+"/components", bytes.NewBufferString(`{"componentCode":"C-01","componentType":"柱"}`))
	component.Header.Set("Content-Type", "application/json")
	missingVersion := httptest.NewRecorder()
	handler.ServeHTTP(missingVersion, component)
	if missingVersion.Code != http.StatusBadRequest || errorCode(t, missingVersion) != "EXPECTED_VERSION_REQUIRED" {
		t.Fatalf("缺少版本错误不正确: status=%d body=%s", missingVersion.Code, missingVersion.Body.String())
	}
}

func errorCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Error.Code
}
