package llmchat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatCompletionsEndpoint(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"https://api.openai.com", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/v1/", "https://api.openai.com/v1/chat/completions"},
		{"http://127.0.0.1:8080/v1/chat/completions", "http://127.0.0.1:8080/v1/chat/completions"},
	}
	for _, tc := range cases {
		if got := chatCompletionsEndpoint(tc.in); got != tc.want {
			t.Fatalf("in=%q got=%q want=%q", tc.in, got, tc.want)
		}
	}
}

func TestCompleteNonStream(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello world"}}]}`))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "sk-test", "gpt-4o-mini", 0)
	text, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello world" {
		t.Fatalf("text=%q", text)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth=%q", gotAuth)
	}
	if gotBody["stream"] != false {
		t.Fatalf("body=%v", gotBody)
	}
}

func TestStreamSSE(t *testing.T) {
	var deltas []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		if body["stream"] != true {
			t.Errorf("want stream true, body=%v", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "", "gpt-4o-mini", 0)
	full, err := c.Stream(context.Background(), []Message{{Role: "user", Content: "x"}}, func(s string) {
		deltas = append(deltas, s)
	})
	if err != nil {
		t.Fatal(err)
	}
	if full != "Hello" {
		t.Fatalf("full=%q", full)
	}
	if strings.Join(deltas, "") != "Hello" || len(deltas) != 2 {
		t.Fatalf("deltas=%v", deltas)
	}
}

func TestStreamFallsBackToComplete(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]any
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		if body["stream"] == true {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`stream broken`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"fallback ok"}}]}`))
	}))
	t.Cleanup(srv.Close)

	var got string
	c := New(srv.URL, "", "m", 0)
	full, err := c.Stream(context.Background(), []Message{{Role: "user", Content: "x"}}, func(s string) {
		got = s
	})
	if err != nil {
		t.Fatal(err)
	}
	if full != "fallback ok" || got != "fallback ok" {
		t.Fatalf("full=%q got=%q calls=%d", full, got, calls)
	}
	if calls != 2 {
		t.Fatalf("calls=%d want 2", calls)
	}
}

func TestConfigured(t *testing.T) {
	t.Setenv(envBaseURL, "")
	if Configured() {
		t.Fatal("want false")
	}
	t.Setenv(envBaseURL, "http://localhost:1")
	if !Configured() {
		t.Fatal("want true")
	}
}
