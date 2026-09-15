package llmchat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	envBaseURL = "ASH_LLM_BASE_URL"
	envAPIKey  = "ASH_LLM_API_KEY"
	envModel   = "ASH_LLM_MODEL"
	envTimeout = "ASH_LLM_TIMEOUT"

	defaultModel   = "gpt-4o-mini"
	defaultTimeout = 60 * time.Second
)

// Message is one OpenAI-compatible chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Client calls an OpenAI-compatible /v1/chat/completions HTTP API.
type Client struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// New builds a client. baseURL may be origin, .../v1, or .../chat/completions.
func New(baseURL, apiKey, model string, timeout time.Duration) *Client {
	baseURL = strings.TrimSpace(baseURL)
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(apiKey),
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

// NewFromEnv reads ASH_LLM_* (caller should ensure BASE_URL is set).
func NewFromEnv() *Client {
	timeout := defaultTimeout
	if v := strings.TrimSpace(os.Getenv(envTimeout)); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			timeout = d
		}
	}
	return New(
		os.Getenv(envBaseURL),
		os.Getenv(envAPIKey),
		os.Getenv(envModel),
		timeout,
	)
}

// Configured reports whether ASH_LLM_BASE_URL is set.
func Configured() bool {
	return strings.TrimSpace(os.Getenv(envBaseURL)) != ""
}

// Model returns the configured model id.
func (c *Client) Model() string {
	if c == nil {
		return ""
	}
	return c.model
}

// Complete posts a non-streaming chat completion and returns assistant text.
func (c *Client) Complete(ctx context.Context, messages []Message) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("llmchat client is nil")
	}
	url := chatCompletionsEndpoint(c.baseURL)
	if url == "" {
		return "", fmt.Errorf("%s is empty", envBaseURL)
	}
	body := map[string]any{
		"model":    c.model,
		"messages": messages,
		"stream":   false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("chat completions HTTP %s: %s", resp.Status, truncateErr(payload))
	}
	var decoded chatCompletionResponse
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return "", err
	}
	text := firstChoiceContent(decoded)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("chat completions response empty")
	}
	return text, nil
}

// Stream posts stream:true and invokes onDelta for each content chunk.
// On stream failure it falls back to Complete.
func (c *Client) Stream(ctx context.Context, messages []Message, onDelta func(string)) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("llmchat client is nil")
	}
	full, err := c.streamOnce(ctx, messages, onDelta)
	if err == nil {
		return full, nil
	}
	text, cerr := c.Complete(ctx, messages)
	if cerr != nil {
		return "", fmt.Errorf("stream failed (%v); complete failed: %w", err, cerr)
	}
	if onDelta != nil && text != "" {
		onDelta(text)
	}
	return text, nil
}

func (c *Client) streamOnce(ctx context.Context, messages []Message, onDelta func(string)) (string, error) {
	url := chatCompletionsEndpoint(c.baseURL)
	if url == "" {
		return "", fmt.Errorf("%s is empty", envBaseURL)
	}
	body := map[string]any{
		"model":    c.model,
		"messages": messages,
		"stream":   true,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("chat completions HTTP %s: %s", resp.Status, truncateErr(payload))
	}
	var b strings.Builder
	sc := bufio.NewScanner(resp.Body)
	// Allow larger SSE frames (default 64K is usually enough; bump for safety).
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			if data == "[DONE]" {
				break
			}
			continue
		}
		var chunk chatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		delta := firstChoiceDelta(chunk)
		if delta == "" {
			continue
		}
		b.WriteString(delta)
		if onDelta != nil {
			onDelta(delta)
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	full := b.String()
	if strings.TrimSpace(full) == "" {
		return "", fmt.Errorf("chat stream empty")
	}
	return full, nil
}

type chatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type chatStreamChunk struct {
	Choices []struct {
		Delta Message `json:"delta"`
	} `json:"choices"`
}

func firstChoiceContent(resp chatCompletionResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Content
}

func firstChoiceDelta(chunk chatStreamChunk) string {
	if len(chunk.Choices) == 0 {
		return ""
	}
	return chunk.Choices[0].Delta.Content
}

// chatCompletionsEndpoint normalizes origin / .../v1 / .../chat/completions into a POST URL.
func chatCompletionsEndpoint(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	base = strings.TrimRight(base, "/")
	lower := strings.ToLower(base)
	if strings.HasSuffix(lower, "/chat/completions") {
		return base
	}
	if strings.HasSuffix(lower, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

func truncateErr(payload []byte) string {
	msg := strings.TrimSpace(string(payload))
	if len(msg) > 256 {
		return msg[:256] + "…"
	}
	return msg
}
