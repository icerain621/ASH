package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	envEmbedBaseURL = "ASH_EMBED_BASE_URL"
	envEmbedAPIKey  = "ASH_EMBED_API_KEY"
	envEmbedModel   = "ASH_EMBED_MODEL"
	envEmbedDim     = "ASH_EMBED_DIM"
	envEmbedTimeout = "ASH_EMBED_TIMEOUT"

	defaultEmbedModel   = "text-embedding-3-small"
	defaultEmbedDim     = 1536
	defaultEmbedTimeout = 30 * time.Second
)

// OpenAICompatEmbedder calls an OpenAI-compatible /v1/embeddings HTTP API.
type OpenAICompatEmbedder struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client

	mu  sync.Mutex
	dim int // configured or learned from first successful response
}

// NewOpenAICompatEmbedder builds a client. baseURL may be origin, .../v1, or .../embeddings.
func NewOpenAICompatEmbedder(baseURL, apiKey, model string, dim int, timeout time.Duration) *OpenAICompatEmbedder {
	baseURL = strings.TrimSpace(baseURL)
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultEmbedModel
	}
	if dim <= 0 {
		dim = defaultEmbedDim
	}
	if timeout <= 0 {
		timeout = defaultEmbedTimeout
	}
	return &OpenAICompatEmbedder{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(apiKey),
		model:   model,
		dim:     dim,
		client:  &http.Client{Timeout: timeout},
	}
}

// NewOpenAICompatEmbedderFromEnv reads ASH_EMBED_* (caller should ensure BASE_URL is set).
func NewOpenAICompatEmbedderFromEnv() *OpenAICompatEmbedder {
	dim := defaultEmbedDim
	if v := strings.TrimSpace(os.Getenv(envEmbedDim)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			dim = n
		}
	}
	timeout := defaultEmbedTimeout
	if v := strings.TrimSpace(os.Getenv(envEmbedTimeout)); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			timeout = d
		}
	}
	return NewOpenAICompatEmbedder(
		os.Getenv(envEmbedBaseURL),
		os.Getenv(envEmbedAPIKey),
		os.Getenv(envEmbedModel),
		dim,
		timeout,
	)
}

// ResolveEmbedder returns OpenAI-compat when ASH_EMBED_BASE_URL is set; otherwise Hash.
func ResolveEmbedder() Embedder {
	if strings.TrimSpace(os.Getenv(envEmbedBaseURL)) == "" {
		return DefaultHashEmbedder()
	}
	return NewOpenAICompatEmbedderFromEnv()
}

// EmbedderKind reports a stable ops label for Profile / diagnostics.
func EmbedderKind(e Embedder) string {
	switch e.(type) {
	case *OpenAICompatEmbedder:
		return "openai_compat"
	case *HashEmbedder:
		return "hash"
	default:
		if e == nil {
			return "none"
		}
		return "custom"
	}
}

func (e *OpenAICompatEmbedder) Dim() int {
	if e == nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dim
}

func (e *OpenAICompatEmbedder) Embed(texts []string) ([][]float32, error) {
	if e == nil || e.client == nil {
		return nil, fmt.Errorf("openai embedder is nil")
	}
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	for _, t := range texts {
		if t == "" {
			return nil, fmt.Errorf("empty text")
		}
	}
	url := embeddingsEndpoint(e.baseURL)
	if url == "" {
		return nil, fmt.Errorf("ASH_EMBED_BASE_URL is empty")
	}
	body := map[string]any{
		"model": e.model,
		"input": texts,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(payload))
		if len(msg) > 256 {
			msg = msg[:256] + "…"
		}
		return nil, fmt.Errorf("embeddings HTTP %s: %s", resp.Status, msg)
	}
	var decoded openaiEmbeddingsResponse
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	if len(decoded.Data) == 0 {
		return nil, fmt.Errorf("embeddings response empty")
	}
	// OpenAI may return out-of-order indices; place by index.
	out := make([][]float32, len(texts))
	for _, item := range decoded.Data {
		if item.Index < 0 || item.Index >= len(out) {
			return nil, fmt.Errorf("embeddings index %d out of range", item.Index)
		}
		if len(item.Embedding) == 0 {
			return nil, fmt.Errorf("empty embedding at index %d", item.Index)
		}
		vec := make([]float32, len(item.Embedding))
		copy(vec, item.Embedding)
		out[item.Index] = vec
	}
	for i, v := range out {
		if len(v) == 0 {
			return nil, fmt.Errorf("missing embedding at index %d", i)
		}
	}
	e.mu.Lock()
	e.dim = len(out[0])
	e.mu.Unlock()
	return out, nil
}

func (e *OpenAICompatEmbedder) String() string {
	if e == nil {
		return "OpenAICompatEmbedder(nil)"
	}
	return fmt.Sprintf("OpenAICompatEmbedder(model=%s,dim=%d)", e.model, e.Dim())
}

type openaiEmbeddingsResponse struct {
	Data []openaiEmbeddingItem `json:"data"`
}

type openaiEmbeddingItem struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// embeddingsEndpoint normalizes origin / .../v1 / .../embeddings into a POST URL.
func embeddingsEndpoint(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	base = strings.TrimRight(base, "/")
	lower := strings.ToLower(base)
	if strings.HasSuffix(lower, "/embeddings") {
		return base
	}
	if strings.HasSuffix(lower, "/v1") {
		return base + "/embeddings"
	}
	return base + "/v1/embeddings"
}

var _ Embedder = (*OpenAICompatEmbedder)(nil)
