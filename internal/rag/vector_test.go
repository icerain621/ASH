package rag

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

// mockVectorStore is an in-memory VectorStore for tests (no live Qdrant).
type mockVectorStore struct {
	mu        sync.Mutex
	available bool
	points    map[string]VectorPoint // key: collection/id
	failUpsert bool
}

func newMockVectorStore(available bool) *mockVectorStore {
	return &mockVectorStore{
		available: available,
		points:    map[string]VectorPoint{},
	}
}

func (m *mockVectorStore) Available() bool {
	if m == nil {
		return false
	}
	return m.available
}

func (m *mockVectorStore) Upsert(space, collection string, points []VectorPoint) error {
	if m.failUpsert {
		return fmt.Errorf("mock upsert failed")
	}
	col := collection
	if col == "" {
		col = CollectionForSpace(space)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range points {
		key := col + "/" + p.ID
		cp := p
		if cp.Payload != nil {
			payload := make(map[string]any, len(cp.Payload))
			for k, v := range cp.Payload {
				payload[k] = v
			}
			cp.Payload = payload
		}
		m.points[key] = cp
	}
	return nil
}

func (m *mockVectorStore) Search(space, collection string, vec []float32, topK int) ([]VectorHit, error) {
	if !m.available {
		return nil, fmt.Errorf("mock store unavailable")
	}
	col := collection
	if col == "" {
		col = CollectionForSpace(space)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	type scored struct {
		hit   VectorHit
		score float32
	}
	var ranked []scored
	for key, p := range m.points {
		if !strings.HasPrefix(key, col+"/") {
			continue
		}
		score := cosineSim(vec, p.Vector)
		ranked = append(ranked, scored{
			hit: VectorHit{
				ID:      p.ID,
				Score:   score,
				Payload: p.Payload,
			},
			score: score,
		})
	}
	// simple descending sort
	for i := 0; i < len(ranked); i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[i].score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}
	if topK <= 0 {
		topK = 10
	}
	if len(ranked) > topK {
		ranked = ranked[:topK]
	}
	out := make([]VectorHit, len(ranked))
	for i, r := range ranked {
		out[i] = r.hit
	}
	return out, nil
}

func cosineSim(a, b []float32) float32 {
	n := len(a)
	if len(b) < n {
		n = len(b)
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
	// Newton's method; fine for tests
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 8; i++ {
		z = 0.5 * (z + x/z)
	}
	return z
}

func TestQueryVectorLaneWhenQdrantMocked(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	mock := newMockVectorStore(true)
	svc := NewService(db).WithEmbedder(DefaultHashEmbedder()).WithVectorStore(mock)

	repo := t.TempDir()
	code := "package p\n\nfunc VectorUniqueSymbol() {\n\t// vector lane target text alpha beta gamma\n}\n"
	if err := os.WriteFile(filepath.Join(repo, "vec.go"), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	idx, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "vec", Embed: true})
	if err != nil {
		t.Fatal(err)
	}
	if idx.Embedded < 1 {
		t.Fatalf("embedded=%d want >=1", idx.Embedded)
	}
	if _, err := svc.RebuildSymbols(RebuildSymbolsRequest{RepoRoot: repo, SpaceID: "vec"}); err != nil {
		t.Fatal(err)
	}

	var refCount int64
	if err := db.Model(&store.RAGVectorRef{}).Where("space_id = ?", "vec").Count(&refCount).Error; err != nil {
		t.Fatal(err)
	}
	if refCount < 1 {
		t.Fatalf("rag_vector_refs=%d want >=1", refCount)
	}

	resp, err := svc.Query(QueryRequest{
		RepoRoot: repo, SpaceID: "vec", Text: "vector lane target alpha", TopK: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RetrievalMode != RetrievalModeHybridVector {
		t.Fatalf("mode=%q want %q", resp.RetrievalMode, RetrievalModeHybridVector)
	}
	if len(resp.Items) == 0 {
		t.Fatal("expected hits including vector lane")
	}
	found := false
	for _, h := range resp.Items {
		if strings.Contains(h.Path, "vec.go") || strings.Contains(h.Snippet, "vector lane") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("items=%+v want vec.go / vector snippet", resp.Items)
	}

	prefer, err := svc.Query(QueryRequest{
		RepoRoot: repo, SpaceID: "vec", Text: "vector lane target alpha", TopK: 5, Prefer: "vector",
	})
	if err != nil {
		t.Fatal(err)
	}
	if prefer.RetrievalMode != RetrievalModeVector {
		t.Fatalf("prefer=vector mode=%q want %q", prefer.RetrievalMode, RetrievalModeVector)
	}
	if len(prefer.Items) == 0 {
		t.Fatal("prefer=vector expected hits")
	}
}

func TestQueryPreferVectorDegradesWithoutHits(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	mock := newMockVectorStore(true)
	svc := NewService(db).WithEmbedder(DefaultHashEmbedder()).WithVectorStore(mock)

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "note.md"), []byte("only text lane content here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "novec", Embed: false}); err != nil {
		t.Fatal(err)
	}

	resp, err := svc.Query(QueryRequest{
		RepoRoot: repo, SpaceID: "novec", Text: "only text lane", TopK: 3, Prefer: "vector",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RetrievalMode == RetrievalModeVector {
		t.Fatalf("mode=%q must degrade when no vector hits", resp.RetrievalMode)
	}
	if len(resp.Items) == 0 {
		t.Fatal("expected text/hybrid hits after degrade")
	}
}

func TestIndexEmbedUsesOpenAICompatEmbedder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[0.5,0.25,0.125]}]}`))
	}))
	defer srv.Close()

	db := store.OpenTest(t, t.TempDir())
	mock := newMockVectorStore(true)
	emb := NewOpenAICompatEmbedder(srv.URL, "k", "test-model", 3, 0)
	svc := NewService(db).WithEmbedder(emb).WithVectorStore(mock)

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "a.md"), []byte("openai embed path\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "oai", Embed: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Embedded < 1 {
		t.Fatalf("embedded=%d", resp.Embedded)
	}
	mock.mu.Lock()
	defer mock.mu.Unlock()
	found := false
	for _, p := range mock.points {
		if len(p.Vector) == 3 && p.Vector[0] == 0.5 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("points=%v want openai dims", mock.points)
	}
	p := svc.Profile("oai")
	if p.EmbedderKind != "openai_compat" {
		t.Fatalf("EmbedderKind=%q", p.EmbedderKind)
	}
}

func TestIndexEmbedOpenAIErrorIsBestEffort(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`fail`))
	}))
	defer srv.Close()

	db := store.OpenTest(t, t.TempDir())
	mock := newMockVectorStore(true)
	emb := NewOpenAICompatEmbedder(srv.URL, "", "m", 8, 0)
	svc := NewService(db).WithEmbedder(emb).WithVectorStore(mock)

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "a.md"), []byte("still index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "fail", Embed: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Documents < 1 {
		t.Fatalf("documents=%d", resp.Documents)
	}
	if resp.Embedded != 0 {
		t.Fatalf("embedded=%d want 0 on embed HTTP error", resp.Embedded)
	}
}

func TestIndexEmbedWritesVectorRefs(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	mock := newMockVectorStore(true)
	svc := NewService(db).WithEmbedder(DefaultHashEmbedder()).WithVectorStore(mock)

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "a.md"), []byte("embed me please\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "emb", Embed: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Embedded < 1 {
		t.Fatalf("embedded=%d", resp.Embedded)
	}
	var refs []store.RAGVectorRef
	if err := db.Where("space_id = ?", "emb").Find(&refs).Error; err != nil {
		t.Fatal(err)
	}
	if len(refs) < 1 {
		t.Fatal("expected vector refs")
	}
	mock.mu.Lock()
	n := len(mock.points)
	mock.mu.Unlock()
	if n < 1 {
		t.Fatal("expected mock upsert points")
	}
}

func TestIndexEmbedSkipWhenStoreDown(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	mock := newMockVectorStore(false)
	svc := NewService(db).WithEmbedder(DefaultHashEmbedder()).WithVectorStore(mock)

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "a.md"), []byte("no embed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "down", Embed: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Documents < 1 {
		t.Fatalf("documents=%d", resp.Documents)
	}
	if resp.Embedded != 0 {
		t.Fatalf("embedded=%d want 0 when store down", resp.Embedded)
	}
}

func TestQuerySucceedsWithoutVectorWhenQdrantDown(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	mock := newMockVectorStore(false)
	svc := NewService(db).WithEmbedder(DefaultHashEmbedder()).WithVectorStore(mock)

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "note.md"), []byte("plain text query works\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "qd"}); err != nil {
		t.Fatal(err)
	}
	// Seed orphaned refs so Query could try vector but store is down.
	now := time.Now().UTC()
	var chunk store.RAGChunk
	if err := db.Where("space_id = ?", "qd").First(&chunk).Error; err != nil {
		t.Fatal(err)
	}
	ref := store.RAGVectorRef{
		ID: "ragvec_orphan", SpaceID: "qd", RepoRoot: chunk.RepoRoot,
		ChunkID: chunk.ID, PointID: "pt_orphan", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&ref).Error; err != nil {
		t.Fatal(err)
	}

	resp, err := svc.Query(QueryRequest{RepoRoot: repo, SpaceID: "qd", Text: "plain text", TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resp.RetrievalMode, "vector") {
		t.Fatalf("mode=%q must not include vector when store down", resp.RetrievalMode)
	}
	if len(resp.Items) == 0 {
		t.Fatal("expected text hits")
	}
}

func TestProfileReportsVectorFields(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	mock := newMockVectorStore(true)
	svc := NewService(db).WithVectorStore(mock)
	now := time.Now().UTC()
	ref := store.RAGVectorRef{
		ID: "ragvec_p", SpaceID: "local", RepoRoot: "/tmp",
		ChunkID: "ragchk_x", PointID: "pt_x", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&ref).Error; err != nil {
		t.Fatal(err)
	}
	p := svc.Profile("local")
	if !p.VectorAvailable {
		t.Fatal("VectorAvailable want true")
	}
	if p.VectorPointCount != 1 {
		t.Fatalf("VectorPointCount=%d want 1", p.VectorPointCount)
	}
	if p.DefaultRetrievalMode != RetrievalModeVector {
		t.Fatalf("DefaultRetrievalMode=%q want %q (vector-only honesty)", p.DefaultRetrievalMode, RetrievalModeVector)
	}
	if p.EmbedderKind == "" {
		t.Fatal("EmbedderKind want non-empty")
	}
	if p.EmbedderDim <= 0 {
		t.Fatalf("EmbedderDim=%d want >0", p.EmbedderDim)
	}
}

func TestProfileReportsHybridVectorWhenBothReady(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	mock := newMockVectorStore(true)
	svc := NewService(db).WithVectorStore(mock)
	now := time.Now().UTC()
	if err := db.Create(&store.RAGPathEntry{
		ID: "ragpath_p", SpaceID: "hv", RepoRoot: "/tmp", Path: "a.go",
		Basename: "a.go", Digest: "d1",
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.RAGVectorRef{
		ID: "ragvec_hv", SpaceID: "hv", RepoRoot: "/tmp",
		ChunkID: "ragchk_hv", PointID: "pt_hv", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	p := svc.Profile("hv")
	if p.DefaultRetrievalMode != RetrievalModeHybridVector {
		t.Fatalf("mode=%q want %q", p.DefaultRetrievalMode, RetrievalModeHybridVector)
	}
}
