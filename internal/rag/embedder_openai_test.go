package rag

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmbeddingsEndpoint(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"https://api.openai.com", "https://api.openai.com/v1/embeddings"},
		{"https://api.openai.com/", "https://api.openai.com/v1/embeddings"},
		{"https://api.openai.com/v1", "https://api.openai.com/v1/embeddings"},
		{"https://api.openai.com/v1/", "https://api.openai.com/v1/embeddings"},
		{"http://127.0.0.1:8080/v1/embeddings", "http://127.0.0.1:8080/v1/embeddings"},
		{"http://local/custom/embeddings", "http://local/custom/embeddings"},
	}
	for _, tc := range cases {
		if got := embeddingsEndpoint(tc.in); got != tc.want {
			t.Fatalf("in=%q got=%q want=%q", tc.in, got, tc.want)
		}
	}
}

func TestOpenAICompatEmbedderEmbed(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":[
				{"index":1,"embedding":[0.2,0.4]},
				{"index":0,"embedding":[0.1,0.3]}
			]
		}`))
	}))
	defer srv.Close()

	e := NewOpenAICompatEmbedder(srv.URL, "sk-test", "text-embedding-3-small", 2, 0)
	out, err := e.Embed([]string{"alpha", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/embeddings" {
		t.Fatalf("path=%q want /v1/embeddings", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth=%q", gotAuth)
	}
	if gotBody["model"] != "text-embedding-3-small" {
		t.Fatalf("model=%v", gotBody["model"])
	}
	if len(out) != 2 || len(out[0]) != 2 || out[0][0] != 0.1 || out[1][0] != 0.2 {
		t.Fatalf("out=%v", out)
	}
	if e.Dim() != 2 {
		t.Fatalf("Dim()=%d want 2 (learned)", e.Dim())
	}
}

func TestOpenAICompatEmbedderHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer srv.Close()

	e := NewOpenAICompatEmbedder(srv.URL, "bad", "m", 8, 0)
	_, err := e.Embed([]string{"x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("err=%v", err)
	}
}

func TestOpenAICompatEmbedderEmptyString(t *testing.T) {
	e := NewOpenAICompatEmbedder("http://example.invalid", "", "m", 8, 0)
	_, err := e.Embed([]string{""})
	if err == nil {
		t.Fatal("expected empty text error")
	}
}

func TestResolveEmbedderHashWhenUnset(t *testing.T) {
	t.Setenv(envEmbedBaseURL, "")
	e := ResolveEmbedder()
	if EmbedderKind(e) != "hash" {
		t.Fatalf("kind=%q want hash", EmbedderKind(e))
	}
}

func TestResolveEmbedderOpenAIWhenURLSet(t *testing.T) {
	t.Setenv(envEmbedBaseURL, "http://127.0.0.1:9")
	t.Setenv(envEmbedModel, "my-model")
	t.Setenv(envEmbedDim, "32")
	e := ResolveEmbedder()
	if EmbedderKind(e) != "openai_compat" {
		t.Fatalf("kind=%q want openai_compat", EmbedderKind(e))
	}
	oe, ok := e.(*OpenAICompatEmbedder)
	if !ok {
		t.Fatal("type assert")
	}
	if oe.model != "my-model" || oe.Dim() != 32 {
		t.Fatalf("model=%q dim=%d", oe.model, oe.Dim())
	}
}

func TestEmbedderKindNil(t *testing.T) {
	if EmbedderKind(nil) != "none" {
		t.Fatal(EmbedderKind(nil))
	}
}
