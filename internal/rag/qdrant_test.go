package rag

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQdrantClientUpsertAndSearch(t *testing.T) {
	var upsertBody string
	mux := http.NewServeMux()
	mux.HandleFunc("/collections", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":{"collections":[]}}`))
	})
	mux.HandleFunc("/collections/ash_test/points", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("upsert method=%s want PUT", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		upsertBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
	})
	mux.HandleFunc("/collections/ash_test/points/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("search method=%s want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":[{"id":"p1","score":0.9,"payload":{"chunkId":"c1","path":"a.go"}}]}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewQdrantClient(srv.URL)
	if !c.Available() {
		t.Fatal("expected Available() true against mock server")
	}

	points := []VectorPoint{{
		ID:     "p1",
		Vector: []float32{0.1, 0.2},
		Payload: map[string]any{
			"chunkId": "c1",
			"path":    "a.go",
			"spaceId": "test",
		},
	}}
	if err := c.Upsert("test", "ash_test", points); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(upsertBody, `"id":"p1"`) {
		t.Fatalf("upsert body missing point id: %s", upsertBody)
	}

	hits, err := c.Search("test", "ash_test", []float32{0.1, 0.2}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "p1" || hits[0].Score != 0.9 {
		t.Fatalf("hits=%+v", hits)
	}
	if hits[0].Payload["chunkId"] != "c1" {
		t.Fatalf("payload=%v", hits[0].Payload)
	}
}

func TestQdrantClientAvailableFalseWhenDown(t *testing.T) {
	c := NewQdrantClient("http://127.0.0.1:1")
	if c.Available() {
		t.Fatal("expected Available() false when server unreachable")
	}
}

func TestSanitizeCollectionName(t *testing.T) {
	if got := CollectionForSpace("my space!"); got != "ash_my_space_" {
		t.Fatalf("got=%q", got)
	}
}

func TestQdrantClientUpsertUsesDefaultCollection(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
	}))
	defer srv.Close()

	c := NewQdrantClient(srv.URL)
	if err := c.Upsert("myspace", "", []VectorPoint{{
		ID:     "x",
		Vector: []float32{1},
	}}); err != nil {
		t.Fatal(err)
	}
	want := "/collections/ash_myspace/points"
	if gotPath != want {
		t.Fatalf("path=%q want %q", gotPath, want)
	}
}

func TestQdrantSearchRequestBody(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collections" {
			w.WriteHeader(http.StatusOK)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":[]}`))
	}))
	defer srv.Close()

	c := NewQdrantClient(srv.URL)
	if _, err := c.Search("s", "col", []float32{0.5, 0.25}, 3); err != nil {
		t.Fatal(err)
	}
	if body["limit"] != float64(3) {
		t.Fatalf("limit=%v want 3", body["limit"])
	}
	vec, ok := body["vector"].([]any)
	if !ok || len(vec) != 2 {
		t.Fatalf("vector=%v", body["vector"])
	}
}
