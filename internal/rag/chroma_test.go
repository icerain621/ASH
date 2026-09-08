package rag

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestChromaClientAgainstGateway(t *testing.T) {
	var mu sync.Mutex
	cols := map[string]string{} // name -> id
	points := map[string][]VectorPoint{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/heartbeat":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"nanosecond heartbeat":1}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/collections":
			body, _ := io.ReadAll(r.Body)
			var req struct {
				Name string `json:"name"`
			}
			_ = json.Unmarshal(body, &req)
			mu.Lock()
			id := "col_" + req.Name
			cols[req.Name] = id
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "name": req.Name})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/upsert"):
			parts := strings.Split(r.URL.Path, "/")
			colID := parts[len(parts)-2]
			body, _ := io.ReadAll(r.Body)
			var req chromaUpsertReq
			_ = json.Unmarshal(body, &req)
			mu.Lock()
			list := points[colID]
			byID := map[string]int{}
			for i, p := range list {
				byID[p.ID] = i
			}
			for i, id := range req.IDs {
				p := VectorPoint{ID: id, Vector: req.Embeddings[i]}
				if i < len(req.Metadatas) {
					p.Payload = req.Metadatas[i]
				}
				if j, ok := byID[id]; ok {
					list[j] = p
				} else {
					list = append(list, p)
				}
			}
			points[colID] = list
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/query"):
			parts := strings.Split(r.URL.Path, "/")
			colID := parts[len(parts)-2]
			body, _ := io.ReadAll(r.Body)
			var req chromaQueryReq
			_ = json.Unmarshal(body, &req)
			mu.Lock()
			list := points[colID]
			mu.Unlock()
			ids := make([]string, 0, len(list))
			dists := make([]float32, 0, len(list))
			metas := make([]map[string]any, 0, len(list))
			for _, p := range list {
				ids = append(ids, p.ID)
				dists = append(dists, 0.1)
				metas = append(metas, p.Payload)
			}
			n := req.NResults
			if n <= 0 || n > len(ids) {
				n = len(ids)
			}
			_ = json.NewEncoder(w).Encode(chromaQueryResp{
				IDs:       [][]string{ids[:n]},
				Distances: [][]float32{dists[:n]},
				Metadatas: [][]map[string]any{metas[:n]},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewChromaClient(srv.URL)
	if !c.Available() {
		t.Fatal("want available against gateway heartbeat")
	}
	vec := []float32{0.1, 0.2, 0.3}
	if err := c.Upsert("space_a", "", []VectorPoint{{
		ID: "chunk_1", Vector: vec, Payload: map[string]any{"path": "a.go"},
	}}); err != nil {
		t.Fatal(err)
	}
	hits, err := c.Search("space_a", "", vec, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "chunk_1" {
		t.Fatalf("hits=%+v", hits)
	}
	if hits[0].Payload["path"] != "a.go" {
		t.Fatalf("payload=%v", hits[0].Payload)
	}
}

func TestResolveChromaUsesClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/heartbeat" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	be := ResolveVectorStoreWith(VectorConfig{Backend: BackendChroma, BaseURL: srv.URL})
	if be == nil || be.Name() != BackendChroma || !be.Available() {
		t.Fatalf("got %+v available=%v", be, be != nil && be.Available())
	}
}
