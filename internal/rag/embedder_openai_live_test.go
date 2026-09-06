//go:build liveembed

package rag

import (
	"os"
	"strings"
	"testing"
)

// TestOpenAICompatEmbedderLive hits a real OpenAI-compatible endpoint.
// Enable: ASH_EMBED_LIVE=1 ASH_EMBED_BASE_URL=... [ASH_EMBED_API_KEY=...] go test -tags=liveembed -run TestOpenAICompatEmbedderLive
func TestOpenAICompatEmbedderLive(t *testing.T) {
	if os.Getenv("ASH_EMBED_LIVE") != "1" {
		t.Skip("ASH_EMBED_LIVE!=1")
	}
	if strings.TrimSpace(os.Getenv("ASH_EMBED_BASE_URL")) == "" {
		t.Fatal("ASH_EMBED_BASE_URL required")
	}
	e := NewOpenAICompatEmbedderFromEnv()
	out, err := e.Embed([]string{"ash live embed probe"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || len(out[0]) < 8 {
		t.Fatalf("unexpected embedding shape: %d x %d", len(out), len(out[0]))
	}
}
