package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	envVectorAPIKey      = "ASH_RAG_VECTOR_API_KEY"
	defaultChromaBaseURL = "http://127.0.0.1:8000"
	chromaProbeTimeout   = 500 * time.Millisecond
	chromaRequestTimeout = 30 * time.Second
)

// ChromaClient talks to a Chroma-compatible HTTP server (local api/v1 shape):
//
//	GET  {base}/api/v1/heartbeat
//	POST {base}/api/v1/collections          (get_or_create)
//	POST {base}/api/v1/collections/{id}/upsert
//	POST {base}/api/v1/collections/{id}/query
//
// Auth: optional X-Chroma-Token from ASH_RAG_VECTOR_API_KEY. Opt-in via
// ASH_RAG_VECTOR_BACKEND=chroma (DX44).
type ChromaClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	mu      sync.Mutex
	colIDs  map[string]string // name -> id
}

// NewChromaClient builds a Chroma-class client. Empty baseURL uses ASH_RAG_VECTOR_URL
// or default http://127.0.0.1:8000.
func NewChromaClient(baseURL string) *ChromaClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = strings.TrimSpace(os.Getenv(envVectorURL))
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultChromaBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &ChromaClient{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(os.Getenv(envVectorAPIKey)),
		client:  &http.Client{Timeout: chromaRequestTimeout},
		colIDs:  map[string]string{},
	}
}

// WithHTTPClient replaces the HTTP client (tests).
func (c *ChromaClient) WithHTTPClient(client *http.Client) *ChromaClient {
	if c == nil {
		return c
	}
	clone := *c
	clone.client = client
	clone.colIDs = map[string]string{}
	return &clone
}

func (c *ChromaClient) Name() string { return BackendChroma }

func (c *ChromaClient) Available() bool {
	if c == nil || c.client == nil {
		return false
	}
	client := c.client
	if client.Timeout == 0 || client.Timeout > chromaProbeTimeout {
		client = &http.Client{Timeout: chromaProbeTimeout}
	}
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/v1/heartbeat", nil)
	if err != nil {
		return false
	}
	c.applyAuth(req)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (c *ChromaClient) UnavailableReason() string {
	if c == nil {
		return "chroma client is nil"
	}
	if !c.Available() {
		return "chroma: set ASH_RAG_VECTOR_URL to a reachable Chroma server (heartbeat)"
	}
	return ""
}

func (c *ChromaClient) applyAuth(req *http.Request) {
	if c == nil || req == nil {
		return
	}
	if c.apiKey != "" {
		req.Header.Set("X-Chroma-Token", c.apiKey)
	}
}

func (c *ChromaClient) resolveCollectionName(space, collection string) string {
	if strings.TrimSpace(collection) != "" {
		return collection
	}
	return CollectionForSpace(space)
}

func (c *ChromaClient) ensureCollection(name string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("chroma client is nil")
	}
	c.mu.Lock()
	if id, ok := c.colIDs[name]; ok && id != "" {
		c.mu.Unlock()
		return id, nil
	}
	c.mu.Unlock()

	body := map[string]any{
		"name":          name,
		"get_or_create": true,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/collections", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("chroma create collection: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var decoded struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		return "", err
	}
	id := strings.TrimSpace(decoded.ID)
	if id == "" {
		id = name // some gateways echo name as id
	}
	c.mu.Lock()
	c.colIDs[name] = id
	c.mu.Unlock()
	return id, nil
}

type chromaUpsertReq struct {
	IDs        []string         `json:"ids"`
	Embeddings [][]float32      `json:"embeddings"`
	Metadatas  []map[string]any `json:"metadatas,omitempty"`
}

func (c *ChromaClient) Upsert(space, collection string, points []VectorPoint) error {
	if c == nil {
		return fmt.Errorf("chroma client is nil")
	}
	if len(points) == 0 {
		return nil
	}
	name := c.resolveCollectionName(space, collection)
	id, err := c.ensureCollection(name)
	if err != nil {
		return err
	}
	reqBody := chromaUpsertReq{
		IDs:        make([]string, len(points)),
		Embeddings: make([][]float32, len(points)),
		Metadatas:  make([]map[string]any, len(points)),
	}
	for i, p := range points {
		reqBody.IDs[i] = p.ID
		reqBody.Embeddings[i] = p.Vector
		reqBody.Metadatas[i] = p.Payload
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/v1/collections/%s/upsert", c.baseURL, id)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma upsert: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

type chromaQueryReq struct {
	QueryEmbeddings [][]float32 `json:"query_embeddings"`
	NResults        int         `json:"n_results"`
	Include         []string    `json:"include,omitempty"`
}

type chromaQueryResp struct {
	IDs       [][]string         `json:"ids"`
	Distances [][]float32        `json:"distances"`
	Metadatas [][]map[string]any `json:"metadatas"`
}

func (c *ChromaClient) Search(space, collection string, vec []float32, topK int) ([]VectorHit, error) {
	if c == nil {
		return nil, fmt.Errorf("chroma client is nil")
	}
	if topK <= 0 {
		topK = 10
	}
	name := c.resolveCollectionName(space, collection)
	id, err := c.ensureCollection(name)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(chromaQueryReq{
		QueryEmbeddings: [][]float32{vec},
		NResults:        topK,
		Include:         []string{"metadatas", "distances"},
	})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/v1/collections/%s/query", c.baseURL, id)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("chroma query: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var decoded chromaQueryResp
	if err := json.Unmarshal(b, &decoded); err != nil {
		return nil, err
	}
	if len(decoded.IDs) == 0 {
		return nil, nil
	}
	ids := decoded.IDs[0]
	hits := make([]VectorHit, 0, len(ids))
	for i, idStr := range ids {
		hit := VectorHit{ID: idStr, Score: 1}
		if len(decoded.Distances) > 0 && i < len(decoded.Distances[0]) {
			// Chroma returns distance; convert to similarity-ish score for ranking.
			d := decoded.Distances[0][i]
			hit.Score = 1 / (1 + d)
		}
		if len(decoded.Metadatas) > 0 && i < len(decoded.Metadatas[0]) {
			hit.Payload = decoded.Metadatas[0][i]
		}
		hits = append(hits, hit)
	}
	return hits, nil
}

var _ VectorBackend = (*ChromaClient)(nil)
