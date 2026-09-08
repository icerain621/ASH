package rag

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
)

const (
	envVectorBackend = "ASH_RAG_VECTOR_BACKEND"
	envVectorURL     = "ASH_RAG_VECTOR_URL"

	BackendQdrant = "qdrant"
	BackendChroma = "chroma"
	BackendMilvus = "milvus"
	BackendMock   = "mock"
)

// VectorBackend extends VectorStore with discovery metadata (v3.0 · DX43).
type VectorBackend interface {
	VectorStore
	Name() string
}

// VectorConfig is loaded from environment (no secrets persisted).
type VectorConfig struct {
	Backend string
	BaseURL string
}

// VectorStatus is a diagnostics snapshot for Scale / Knowledge (DX47).
type VectorStatus struct {
	Backend   string `json:"backend"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// VectorConfigFromEnv reads ASH_RAG_VECTOR_BACKEND / URL.
// Default backend is qdrant (preserves DX17–DX26 behavior).
func VectorConfigFromEnv() VectorConfig {
	cfg := VectorConfig{
		Backend: strings.ToLower(strings.TrimSpace(os.Getenv(envVectorBackend))),
		BaseURL: strings.TrimSpace(os.Getenv(envVectorURL)),
	}
	if cfg.Backend == "" {
		cfg.Backend = BackendQdrant
	}
	return cfg
}

// ResolveVectorStore picks a VectorBackend from env (DX43+).
// mock is always Available; qdrant remains default.
func ResolveVectorStore() VectorBackend {
	return ResolveVectorStoreWith(VectorConfigFromEnv())
}

// ResolveVectorStoreWith picks a backend for cfg.
func ResolveVectorStoreWith(cfg VectorConfig) VectorBackend {
	switch strings.ToLower(strings.TrimSpace(cfg.Backend)) {
	case BackendMock:
		return NewMockVectorStore()
	case BackendChroma:
		url := cfg.BaseURL
		if url == "" {
			url = strings.TrimSpace(os.Getenv(envVectorURL))
		}
		return NewChromaClient(url)
	case BackendMilvus:
		url := cfg.BaseURL
		if url == "" {
			url = strings.TrimSpace(os.Getenv(envVectorURL))
		}
		return NewMilvusClient(url)
	case BackendQdrant, "":
		url := cfg.BaseURL
		if url == "" {
			url = DefaultQdrantURL()
		}
		return namedQdrant{QdrantClient: NewQdrantClient(url)}
	default:
		return newUnavailableVector(cfg.Backend, "unknown vector backend")
	}
}

// ProbeVectorStatus reports configured backend readiness.
func ProbeVectorStatus() VectorStatus {
	cfg := VectorConfigFromEnv()
	be := ResolveVectorStoreWith(cfg)
	st := VectorStatus{Backend: cfg.Backend}
	if be == nil {
		st.Reason = "no backend"
		return st
	}
	st.Backend = be.Name()
	st.Available = be.Available()
	if !st.Available {
		if u, ok := be.(interface{ UnavailableReason() string }); ok {
			st.Reason = u.UnavailableReason()
		} else {
			st.Reason = "backend unavailable"
		}
	}
	return st
}

// namedQdrant wraps QdrantClient with Name().
type namedQdrant struct {
	*QdrantClient
}

func (n namedQdrant) Name() string { return BackendQdrant }

// MockVectorStore is an in-memory VectorBackend for contract tests / dry-run.
type MockVectorStore struct {
	mu     sync.Mutex
	points map[string][]VectorPoint // collection -> points
}

// NewMockVectorStore returns an always-available in-memory store.
func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{points: map[string][]VectorPoint{}}
}

func (m *MockVectorStore) Name() string { return BackendMock }

func (m *MockVectorStore) Available() bool { return m != nil }

func (m *MockVectorStore) Upsert(space, collection string, points []VectorPoint) error {
	if m == nil {
		return fmt.Errorf("mock vector store is nil")
	}
	col := collection
	if strings.TrimSpace(col) == "" {
		col = CollectionForSpace(space)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	existing := m.points[col]
	byID := map[string]int{}
	for i, p := range existing {
		byID[p.ID] = i
	}
	for _, p := range points {
		if i, ok := byID[p.ID]; ok {
			existing[i] = p
		} else {
			existing = append(existing, p)
			byID[p.ID] = len(existing) - 1
		}
	}
	m.points[col] = existing
	return nil
}

func (m *MockVectorStore) Search(space, collection string, vec []float32, topK int) ([]VectorHit, error) {
	if m == nil {
		return nil, fmt.Errorf("mock vector store is nil")
	}
	if topK <= 0 {
		topK = 8
	}
	col := collection
	if strings.TrimSpace(col) == "" {
		col = CollectionForSpace(space)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	pts := m.points[col]
	out := make([]VectorHit, 0, len(pts))
	for _, p := range pts {
		score := cosineSim(vec, p.Vector)
		out = append(out, VectorHit{ID: p.ID, Score: score, Payload: p.Payload})
	}
	// naive topK by score
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Score > out[i].Score {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > topK {
		out = out[:topK]
	}
	return out, nil
}

func cosineSim(a, b []float32) float32 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return 0
	}
	var dot, na, nb float32
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (sqrt32(na) * sqrt32(nb))
}

func sqrt32(x float32) float32 {
	if x <= 0 {
		return 0
	}
	return float32(math.Sqrt(float64(x)))
}

type unavailableVector struct {
	name   string
	reason string
}

func newUnavailableVector(name, reason string) *unavailableVector {
	return &unavailableVector{name: name, reason: reason}
}

func (u *unavailableVector) Name() string              { return u.name }
func (u *unavailableVector) Available() bool           { return false }
func (u *unavailableVector) UnavailableReason() string { return u.reason }
func (u *unavailableVector) Upsert(string, string, []VectorPoint) error {
	return fmt.Errorf("vector backend %s unavailable: %s", u.name, u.reason)
}
func (u *unavailableVector) Search(string, string, []float32, int) ([]VectorHit, error) {
	return nil, fmt.Errorf("vector backend %s unavailable: %s", u.name, u.reason)
}
