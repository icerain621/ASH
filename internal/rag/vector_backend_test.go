package rag

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestVectorConfigDefaultQdrant(t *testing.T) {
	t.Setenv(envVectorBackend, "")
	t.Setenv(envVectorURL, "")
	cfg := VectorConfigFromEnv()
	if cfg.Backend != BackendQdrant {
		t.Fatalf("backend=%q want qdrant", cfg.Backend)
	}
	be := ResolveVectorStoreWith(cfg)
	if be == nil || be.Name() != BackendQdrant {
		t.Fatalf("got %+v", be)
	}
}

func TestResolveVectorMockAvailable(t *testing.T) {
	be := ResolveVectorStoreWith(VectorConfig{Backend: BackendMock})
	if be == nil || !be.Available() || be.Name() != BackendMock {
		t.Fatalf("mock want available, got %+v", be)
	}
	vec := []float32{1, 0, 0}
	if err := be.Upsert("local", "", []VectorPoint{{ID: "p1", Vector: vec, Payload: map[string]any{"path": "a.go"}}}); err != nil {
		t.Fatal(err)
	}
	hits, err := be.Search("local", "", vec, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "p1" {
		t.Fatalf("hits=%+v", hits)
	}
}

func TestResolveChromaMilvusStubUnavailable(t *testing.T) {
	// Unreachable URLs → Available=false for both real clients.
	be := ResolveVectorStoreWith(VectorConfig{Backend: BackendChroma, BaseURL: "http://127.0.0.1:1"})
	if be == nil || be.Name() != BackendChroma || be.Available() {
		t.Fatalf("chroma unreachable: %+v available=%v", be, be != nil && be.Available())
	}
	be = ResolveVectorStoreWith(VectorConfig{Backend: BackendMilvus, BaseURL: "http://127.0.0.1:1"})
	if be == nil || be.Name() != BackendMilvus || be.Available() {
		t.Fatalf("milvus unreachable: %+v available=%v", be, be != nil && be.Available())
	}
}

func TestProbeVectorStatusUnknown(t *testing.T) {
	t.Setenv(envVectorBackend, "weird")
	st := ProbeVectorStatus()
	if st.Available || st.Backend != "weird" {
		t.Fatalf("status=%+v", st)
	}
	if st.Reason == "" {
		t.Fatal("want reason")
	}
}

func TestProfileReportsVectorBackend(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db).WithVectorStore(NewMockVectorStore())
	p := svc.Profile("local")
	if p.VectorBackend != BackendMock {
		t.Fatalf("vectorBackend=%q want mock", p.VectorBackend)
	}
	if !p.VectorAvailable {
		t.Fatal("want vectorAvailable")
	}
}
