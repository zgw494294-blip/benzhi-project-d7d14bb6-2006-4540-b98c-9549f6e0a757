package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

type selfcheckClient struct {
	base   string
	client *http.Client
}

func runSelfcheck(ctx context.Context, addr string, server *http.Server) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("selfcheck监听失败: %w", err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	c := selfcheckClient{base: "http://" + addr, client: &http.Client{Timeout: 3 * time.Second}}
	dossier, err := c.post(ctx, "/api/v1/dossiers", map[string]any{
		"buildingCode": "BJ-001", "title": "正殿木构", "surveyBoundary": "正殿及东次间",
	}, 0, "self-create", "勘察工程师")
	if err != nil {
		return err
	}
	id := stringField(dossier, "id")
	version := uintField(dossier, "version")
	components, err := c.post(ctx, "/api/v1/dossiers/"+id+"/components", map[string]any{
		"components": []map[string]any{{"componentCode": "C-01", "componentType": "柱", "location": "东次间", "requiredChecks": []string{"裂缝"}}},
	}, version, "self-components", "勘察工程师")
	if err != nil {
		return err
	}
	version = uintField(components, "version")
	componentID, err := firstMapKey(components, "components")
	if err != nil {
		return err
	}
	observed, err := c.post(ctx, "/api/v1/dossiers/"+id+"/observations", map[string]any{
		"componentID": componentID, "conditionType": "裂缝", "locationDetail": "柱身东侧",
		"severity": "LOW", "measurements": map[string]float64{"length": 1},
		"evidenceRefs": []string{"photo-1"}, "observedAt": time.Now().UTC().Add(-time.Minute),
	}, version, "self-observe", "勘察工程师")
	if err != nil {
		return err
	}
	version = uintField(observed, "version")
	observationID, err := latestObservationID(observed, componentID)
	if err != nil {
		return err
	}
	assessed, err := c.post(ctx, "/api/v1/dossiers/"+id+"/assess", map[string]any{}, version, "self-assess", "校核工程师")
	if err != nil {
		return err
	}
	dossier = objectField(assessed, "dossier")
	version = uintField(dossier, "version")
	action := map[string]any{
		"componentID": componentID, "action": "灌浆修补",
		"materialConstraint": "使用与原构件相容的木材和灌浆料", "acceptanceStandard": "裂缝宽度稳定",
	}
	planOne, err := c.post(ctx, "/api/v1/dossiers/"+id+"/plans", map[string]any{
		"revision": 1, "referencedObservationIDs": []string{observationID}, "actions": []map[string]any{action},
	}, version, "self-plan-1", "方案编制人")
	if err != nil {
		return err
	}
	version = uintField(planOne, "version")
	returned, err := c.post(ctx, "/api/v1/dossiers/"+id+"/reviews", map[string]any{
		"decision": "RETURN", "comments": []string{"补充榫卯节点验收记录"},
	}, version, "self-return", "责任审核员")
	if err != nil {
		return err
	}
	if stringField(returned, "state") != "CHANGES_REQUESTED" {
		return fmt.Errorf("退回后状态不正确")
	}
	version = uintField(returned, "version")
	reviewFindingID, err := reviewFindingID(returned)
	if err != nil {
		return err
	}
	planTwo, err := c.post(ctx, "/api/v1/dossiers/"+id+"/plans", map[string]any{
		"revision": 2, "referencedObservationIDs": []string{observationID},
		"resolvedFindingIDs": []string{reviewFindingID}, "actions": []map[string]any{action},
		"acceptanceCriteria": []string{"榫卯节点记录完整并经审核员签认"},
	}, version, "self-plan-2", "方案编制人")
	if err != nil {
		return err
	}
	version = uintField(planTwo, "version")
	approved, err := c.post(ctx, "/api/v1/dossiers/"+id+"/reviews", map[string]any{
		"decision": "APPROVE", "comments": []string{"修订内容完整"},
	}, version, "self-approve", "责任审核员")
	if err != nil {
		return err
	}
	version = uintField(approved, "version")
	frozen, err := c.post(ctx, "/api/v1/dossiers/"+id+"/freeze", map[string]any{}, version, "self-freeze", "责任审核员")
	if err != nil {
		return err
	}
	version = uintField(frozen, "version")
	if _, err := c.post(ctx, "/api/v1/dossiers/"+id+"/release", map[string]any{}, version, "self-release", "放行签发人"); err != nil {
		return err
	}
	for _, path := range []string{"/timeline", "/risk", "/certificate"} {
		if _, err := c.get(ctx, "/api/v1/dossiers/"+id+path); err != nil {
			return err
		}
	}
	select {
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	default:
	}
	return nil
}

func (c selfcheckClient) post(ctx context.Context, path string, body any, version uint64, key, actor string) (map[string]any, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor", actor)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	if version > 0 {
		req.Header.Set("X-Expected-Version", strconv.FormatUint(version, 10))
	}
	return c.do(req)
}

func (c selfcheckClient) get(ctx context.Context, path string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c selfcheckClient) do(req *http.Request) (map[string]any, error) {
	response, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%s响应JSON无效: %w", req.URL.Path, err)
	}
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("%s返回%d: %v", req.URL.Path, response.StatusCode, result)
	}
	return result, nil
}

func stringField(value map[string]any, key string) string {
	result, _ := value[key].(string)
	return result
}

func uintField(value map[string]any, key string) uint64 {
	result, _ := value[key].(float64)
	return uint64(result)
}

func objectField(value map[string]any, key string) map[string]any {
	result, _ := value[key].(map[string]any)
	return result
}

func firstMapKey(value map[string]any, key string) (string, error) {
	items, ok := value[key].(map[string]any)
	if !ok || len(items) == 0 {
		return "", fmt.Errorf("%s响应缺少%s", key, key)
	}
	for id := range items {
		return id, nil
	}
	return "", fmt.Errorf("%s响应为空", key)
}

func latestObservationID(value map[string]any, componentID string) (string, error) {
	observations, ok := value["observations"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("观察响应缺少observations")
	}
	items, ok := observations[componentID].([]any)
	if !ok || len(items) == 0 {
		return "", fmt.Errorf("观察响应缺少构件修订")
	}
	latest, ok := items[len(items)-1].(map[string]any)
	if !ok || stringField(latest, "id") == "" {
		return "", fmt.Errorf("观察响应缺少id")
	}
	return stringField(latest, "id"), nil
}

func reviewFindingID(value map[string]any) (string, error) {
	findings, ok := value["findings"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("退回响应缺少findings")
	}
	for id, raw := range findings {
		finding, _ := raw.(map[string]any)
		if stringField(finding, "source") == "REVIEW" {
			return id, nil
		}
	}
	return "", fmt.Errorf("退回未生成审核问题项")
}
