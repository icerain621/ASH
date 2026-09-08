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
	defaultMilvusBaseURL = "http://127.0.0.1:19530"
	milvusProbeTimeout   = 500 * time.Millisecond
	milvusRequestTimeout = 30 * time.Second
	milvusVectorField    = "vector"
	milvusIDField        = "id"
)

// MilvusClient talks to a Milvus-compatible REST v2 control plane:
//
//	POST {base}/v2/vectordb/collections/list
//	POST {base}/v2/vectordb/collections/create
//	POST {base}/v2/vectordb/entities/upsert
//	POST {base}/v2/vectordb/entities/search
//
// Auth: optional Authorization Bearer from ASH_RAG_VECTOR_API_KEY.
// Opt-in via ASH_RAG_VECTOR_BACKEND=milvus (DX45).
type MilvusClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	mu      sync.Mutex
	ensured map[string]bool
}

// NewMilvusClient builds a Milvus-class client. Empty baseURL uses ASH_RAG_VECTOR_URL
// or default http://127.0.0.1:19530.
func NewMilvusClient(baseURL string) *MilvusClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = strings.TrimSpace(os.Getenv(envVectorURL))
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultMilvusBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &MilvusClient{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(os.Getenv(envVectorAPIKey)),
		client:  &http.Client{Timeout: milvusRequestTimeout},
		ensured: map[string]bool{},
	}
}

// WithHTTPClient replaces the HTTP client (tests).
func (c *MilvusClient) WithHTTPClient(client *http.Client) *MilvusClient {
	if c == nil {
		return c
	}
	clone := *c
	clone.client = client
	clone.ensured = map[string]bool{}
	return &clone
}

func (c *MilvusClient) Name() string { return BackendMilvus }

func (c *MilvusClient) Available() bool {
	if c == nil || c.client == nil {
		return false
	}
	client := c.client
	if client.Timeout == 0 || client.Timeout > milvusProbeTimeout {
		client = &http.Client{Timeout: milvusProbeTimeout}
	}
	// Prefer collections/list (REST v2); fall back to /healthz for slim gateways.
	if c.probePOST(client, "/v2/vectordb/collections/list", map[string]any{"dbName": "_default"}) {
		return true
	}
	return c.probeGET(client, "/healthz")
}

func (c *MilvusClient) UnavailableReason() string {
	if c == nil {
		return "milvus client is nil"
	}
	if !c.Available() {
		return "milvus: set ASH_RAG_VECTOR_URL to a reachable Milvus REST endpoint"
	}
	return ""
}

func (c *MilvusClient) applyAuth(req *http.Request) {
	if c == nil || req == nil || c.apiKey == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
}

func (c *MilvusClient) probeGET(client *http.Client, path string) bool {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
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

func (c *MilvusClient) probePOST(client *http.Client, path string, body any) bool {
	raw, err := json.Marshal(body)
	if err != nil {
		return false
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	var decoded struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		// Some gateways return empty/non-JSON OK bodies.
		return true
	}
	return decoded.Code == 0
}

func (c *MilvusClient) resolveCollectionName(space, collection string) string {
	if strings.TrimSpace(collection) != "" {
		return collection
	}
	return CollectionForSpace(space)
}

func (c *MilvusClient) postJSON(path string, body any) ([]byte, int, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

func (c *MilvusClient) ensureCollection(name string, dim int) error {
	if c == nil {
		return fmt.Errorf("milvus client is nil")
	}
	c.mu.Lock()
	if c.ensured[name] {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	if dim <= 0 {
		dim = DefaultHashEmbedderDim
	}
	body := map[string]any{
		"collectionName": name,
		"dimension":      dim,
		"metricType":     "COSINE",
		"idType":         "VarChar",
		"autoID":         false,
	}
	b, status, err := c.postJSON("/v2/vectordb/collections/create", body)
	if err != nil {
		return err
	}
	if status >= 200 && status < 300 {
		var decoded struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(b, &decoded)
		msg := strings.ToLower(decoded.Message + string(b))
		if decoded.Code == 0 || strings.Contains(msg, "already") || strings.Contains(msg, "exist") {
			c.mu.Lock()
			c.ensured[name] = true
			c.mu.Unlock()
			return nil
		}
		if decoded.Code != 0 {
			return fmt.Errorf("milvus create collection: code=%d %s", decoded.Code, strings.TrimSpace(string(b)))
		}
		c.mu.Lock()
		c.ensured[name] = true
		c.mu.Unlock()
		return nil
	}
	msg := strings.ToLower(string(b))
	if status == http.StatusConflict || strings.Contains(msg, "already") || strings.Contains(msg, "exist") {
		c.mu.Lock()
		c.ensured[name] = true
		c.mu.Unlock()
		return nil
	}
	return fmt.Errorf("milvus create collection: %d: %s", status, strings.TrimSpace(string(b)))
}

func (c *MilvusClient) Upsert(space, collection string, points []VectorPoint) error {
	if c == nil {
		return fmt.Errorf("milvus client is nil")
	}
	if len(points) == 0 {
		return nil
	}
	name := c.resolveCollectionName(space, collection)
	dim := len(points[0].Vector)
	if err := c.ensureCollection(name, dim); err != nil {
		return err
	}
	data := make([]map[string]any, len(points))
	for i, p := range points {
		row := map[string]any{
			milvusIDField:     p.ID,
			milvusVectorField: p.Vector,
		}
		for k, v := range p.Payload {
			if k == milvusIDField || k == milvusVectorField {
				continue
			}
			row[k] = v
		}
		data[i] = row
	}
	body := map[string]any{
		"collectionName": name,
		"data":           data,
	}
	b, status, err := c.postJSON("/v2/vectordb/entities/upsert", body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("milvus upsert: %d: %s", status, strings.TrimSpace(string(b)))
	}
	var decoded struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(b, &decoded); err == nil && decoded.Code != 0 {
		return fmt.Errorf("milvus upsert: code=%d %s", decoded.Code, strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *MilvusClient) Search(space, collection string, vec []float32, topK int) ([]VectorHit, error) {
	if c == nil {
		return nil, fmt.Errorf("milvus client is nil")
	}
	if topK <= 0 {
		topK = 10
	}
	name := c.resolveCollectionName(space, collection)
	if err := c.ensureCollection(name, len(vec)); err != nil {
		return nil, err
	}
	body := map[string]any{
		"collectionName": name,
		"data":           [][]float32{vec},
		"annsField":      milvusVectorField,
		"limit":          topK,
		"outputFields":   []string{"*"},
	}
	b, status, err := c.postJSON("/v2/vectordb/entities/search", body)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("milvus search: %d: %s", status, strings.TrimSpace(string(b)))
	}
	var decoded struct {
		Code int                `json:"code"`
		Data [][]map[string]any `json:"data"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		return nil, err
	}
	if decoded.Code != 0 {
		return nil, fmt.Errorf("milvus search: code=%d %s", decoded.Code, strings.TrimSpace(string(b)))
	}
	if len(decoded.Data) == 0 {
		return nil, nil
	}
	rows := decoded.Data[0]
	hits := make([]VectorHit, 0, len(rows))
	for _, row := range rows {
		hit := VectorHit{Payload: map[string]any{}}
		if id, ok := row[milvusIDField]; ok {
			hit.ID = fmt.Sprint(id)
		}
		if d, ok := row["distance"].(float64); ok {
			hit.Score = float32(1 / (1 + d))
		} else if s, ok := row["score"].(float64); ok {
			hit.Score = float32(s)
		} else {
			hit.Score = 1
		}
		for k, v := range row {
			if k == milvusIDField || k == milvusVectorField || k == "distance" || k == "score" {
				continue
			}
			hit.Payload[k] = v
		}
		hits = append(hits, hit)
	}
	return hits, nil
}

var _ VectorBackend = (*MilvusClient)(nil)
