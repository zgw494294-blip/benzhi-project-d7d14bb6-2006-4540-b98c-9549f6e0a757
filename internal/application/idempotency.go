package application

import (
	"strings"
	"timber-release-gate/internal/domain"
)

// requestFingerprint keeps idempotency payload construction consistent across
// commands.
func requestFingerprint(scope, actor string, payload any) string {
	return domain.Digest(struct {
		Scope   string
		Payload any
	}{scope, payload})
}

type IdempotencyStatus struct {
	Key       string `json:"key"`
	Present   bool   `json:"present"`
	DossierID string `json:"dossierID,omitempty"`
}

func (s *Service) CheckIdempotency(key, payload string) (IdempotencyStatus, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return IdempotencyStatus{}, domain.Invalid("IDEMPOTENCY_REQUIRED", "写请求必须提供幂等键")
	}
	d, ok, e := s.st.Idempotent(key, domain.Digest(payload))
	if e != nil {
		return IdempotencyStatus{}, e
	}
	out := IdempotencyStatus{Key: key, Present: ok}
	if d != nil {
		out.DossierID = d.ID
	}
	return out, nil
}
func NormalizeActor(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "anonymous"
	}
	return actor
}
func NormalizeKey(key string) string { return strings.TrimSpace(key) }
