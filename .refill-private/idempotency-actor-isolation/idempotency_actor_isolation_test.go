package idempotency_actor_isolation_test

import (
	"testing"

	"timber-release-gate/internal/application"
	"timber-release-gate/internal/domain"
	"timber-release-gate/internal/store"
)

func TestIdempotencyKeyCannotReplayAcrossActors(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	svc := application.New(st)
	in := application.CreateInput{BuildingCode: "BJ-001", Title: "东次间", SurveyBoundary: "正殿"}
	first, err := svc.Create(in, "same-key", "审核员甲")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := svc.Create(in, "same-key", "审核员乙")
	if err == nil {
		t.Fatalf("expected idempotency conflict, got dossier %s", second.ID)
	}
	de, ok := err.(*domain.Error)
	if !ok || de.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("expected IDEMPOTENCY_CONFLICT, got %T %v", err, err)
	}
	if second != nil && second.ID == first.ID {
		t.Fatalf("cross-actor retry returned the first dossier")
	}
}
