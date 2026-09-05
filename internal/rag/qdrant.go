package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	defaultQdrantURL      = "http://127.0.0.1:6333"
	qdrantProbeTimeout    = 500 * time.Millisecond
	qdrantRequestTimeout  = 30 * time.Second
	envQdrantURL          = "ASH_QDRANT_URL"
)

// VectorPoint is a single upsert target for VectorStore.
type VectorPoint struct {
	ID      string
	Vector  []float32
	Payload map[string]any // e.g. chunkId, path, spaceId
}

// VectorHit is a single search result from VectorStore.
type VectorHit struct {
	ID      string
	Score   float32
	Payload map[string]any
}

// VectorStore persists and queries dense vectors (e.g. Qdrant).
type VectorStore interface {
	Upsert(space, collection string, points []VectorPoint) error
	Search(space, collection string, vec []float32, topK int) ([]VectorHit, error)
	Available() bool
}

// QdrantClient implements VectorStore via Qdrant REST JSON API.
type QdrantClient struct {
	baseURL string
	client  *http.Client
}

// NewQdrantClient returns a client for baseURL, or ASH_QDRANT_URL / default when empty.
func NewQdrantClient(baseURL string) *QdrantClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = os.Getenv(envQdrantURL)
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultQdrantURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &QdrantClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: qdrantRequestTimeout,
		},
	}
}

// WithHTTPClient replaces the HTTP client (for tests).
func (c *QdrantClient) WithHTTPClient(client *http.Client) *QdrantClient {
	if c == nil {
		return c
	}
	clone := *c
	clone.client = client
	return &clone
}

var collectionSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// CollectionForSpace returns a Qdrant-safe collection name for a space id.
func CollectionForSpace(space string) string {
	safe := collectionSanitizer.ReplaceAllString(strings.TrimSpace(space), "_")
	if safe == "" {
		safe = "default"
	}
	return "ash_" + safe
}

func (c *QdrantClient) resolveCollection(space, collection string) string {
	if strings.TrimSpace(collection) != "" {
		return collection
	}
	return CollectionForSpace(space)
}

// Available probes Qdrant with a cheap GET /collections; false on dial/error, never panics.
func (c *QdrantClient) Available() bool {
	if c == nil || c.client == nil {
		return false
	}
	client := c.client
	if client.Timeout == 0 || client.Timeout > qdrantProbeTimeout {
		client = &http.Client{Timeout: qdrantProbeTimeout}
	}
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/collections", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

type qdrantUpsertRequest struct {
	Points []qdrantPoint `json:"points"`
}

type qdrantPoint struct {
	ID      any                `json:"id"`
	Vector  []float32          `json:"vector"`
	Payload map[string]any     `json:"payload,omitempty"`
}

type qdrantSearchRequest struct {
	Vector      []float32 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
}

type qdrantSearchResponse struct {
	Result []qdrantSearchHit `json:"result"`
}

type qdrantSearchHit struct {
	ID      any            `json:"id"`
	Score   float32        `json:"score"`
	Payload map[string]any `json:"payload"`
}

// Upsert PUTs points into the collection.
func (c *QdrantClient) Upsert(space, collection string, points []VectorPoint) error {
	if c == nil {
		return fmt.Errorf("qdrant client is nil")
	}
	if len(points) == 0 {
		return nil
	}
	col := c.resolveCollection(space, collection)
	body := qdrantUpsertRequest{Points: make([]qdrantPoint, len(points))}
	for i, p := range points {
		body.Points[i] = qdrantPoint{
			ID:      p.ID,
			Vector:  p.Vector,
			Payload: p.Payload,
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/collections/%s/points?wait=true", c.baseURL, col)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant upsert: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

// Search POSTs a vector query and returns hits.
func (c *QdrantClient) Search(space, collection string, vec []float32, topK int) ([]VectorHit, error) {
	if c == nil {
		return nil, fmt.Errorf("qdrant client is nil")
	}
	if topK <= 0 {
		topK = 10
	}
	col := c.resolveCollection(space, collection)
	body := qdrantSearchRequest{
		Vector:      vec,
		Limit:       topK,
		WithPayload: true,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, col)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant search: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var decoded qdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	hits := make([]VectorHit, len(decoded.Result))
	for i, r := range decoded.Result {
		hits[i] = VectorHit{
			ID:      fmt.Sprint(r.ID),
			Score:   r.Score,
			Payload: r.Payload,
		}
	}
	return hits, nil
}

// Ensure QdrantClient implements VectorStore.
var _ VectorStore = (*QdrantClient)(nil)
