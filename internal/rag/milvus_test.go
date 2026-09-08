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

func TestMilvusClientAgainstGateway(t *testing.T) {
	var mu sync.Mutex
	cols := map[string]bool{}
	points := map[string][]map[string]any{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/vectordb/collections/list":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []string{}})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/vectordb/collections/create":
			var req struct {
				CollectionName string `json:"collectionName"`
			}
			_ = json.Unmarshal(body, &req)
			mu.Lock()
			cols[req.CollectionName] = true
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/vectordb/entities/upsert":
			var req struct {
				CollectionName string           `json:"collectionName"`
				Data           []map[string]any `json:"data"`
			}
			_ = json.Unmarshal(body, &req)
			mu.Lock()
			points[req.CollectionName] = append(points[req.CollectionName], req.Data...)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/vectordb/entities/search":
			var req struct {
				CollectionName string `json:"collectionName"`
				Limit          int    `json:"limit"`
			}
			_ = json.Unmarshal(body, &req)
			mu.Lock()
			rows := points[req.CollectionName]
			mu.Unlock()
			out := make([]map[string]any, 0, len(rows))
			for _, row := range rows {
				item := map[string]any{"distance": 0.05}
				for k, v := range row {
					item[k] = v
				}
				out = append(out, item)
			}
			if req.Limit > 0 && len(out) > req.Limit {
				out = out[:req.Limit]
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []any{out}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewMilvusClient(srv.URL)
	if !c.Available() {
		t.Fatal("want available against collections/list")
	}
	vec := []float32{0.1, 0.2, 0.3}
	if err := c.Upsert("space_m", "", []VectorPoint{{
		ID: "chunk_m1", Vector: vec, Payload: map[string]any{"path": "m.go"},
	}}); err != nil {
		t.Fatal(err)
	}
	hits, err := c.Search("space_m", "", vec, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "chunk_m1" {
		t.Fatalf("hits=%+v", hits)
	}
	if hits[0].Payload["path"] != "m.go" {
		t.Fatalf("payload=%v", hits[0].Payload)
	}
	if !strings.HasPrefix(CollectionForSpace("space_m"), "ash_") {
		t.Fatal("collection naming")
	}
}

func TestResolveMilvusUsesClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/vectordb/collections/list" {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	be := ResolveVectorStoreWith(VectorConfig{Backend: BackendMilvus, BaseURL: srv.URL})
	if be == nil || be.Name() != BackendMilvus || !be.Available() {
		t.Fatalf("got %+v available=%v", be, be != nil && be.Available())
	}
}

func TestMilvusUnavailableOnBadURL(t *testing.T) {
	c := NewMilvusClient("http://127.0.0.1:1")
	if c.Available() {
		t.Fatal("unreachable milvus should be unavailable")
	}
}
