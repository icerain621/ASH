package toolbus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

func registerMCPTools(r *Registry) {
	r.Register("mcp.call", RiskMedium, mcpCall)
}

func mcpCall(_ Context, args map[string]any) (map[string]any, error) {
	if err := validateMCPTopLevelArgs(args); err != nil {
		return nil, err
	}
	serverURL := firstStringArg(args, "serverURL", "endpoint", "url")
	toolName := firstStringArg(args, "name", "tool", "toolName")
	if strings.TrimSpace(serverURL) == "" {
		return nil, mcpError{class: "schema", msg: "serverURL is required"}
	}
	if err := validateMCPServerURL(serverURL); err != nil {
		return nil, err
	}
	if strings.TrimSpace(toolName) == "" {
		return nil, mcpError{class: "schema", msg: "tool name is required"}
	}
	timeoutMs := int64Arg(args, "timeoutMs", 30000)
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if timeoutMs > 120000 {
		return nil, mcpError{class: "policy", msg: fmt.Sprintf("timeoutMs %d exceeds max 120000", timeoutMs)}
	}
	input := map[string]any{}
	if v, ok := args["arguments"].(map[string]any); ok {
		input = v
	} else if v, ok := args["input"].(map[string]any); ok {
		input = v
	} else if v, ok := args["args"].(map[string]any); ok {
		input = v
	}
	if err := validateMCPArgumentWhitelist(args, input); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      "ash-mcp-" + toolName,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": input,
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, mcpError{class: "schema", msg: err.Error()}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL, bytes.NewReader(body))
	if err != nil {
		return nil, mcpError{class: "schema", msg: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, mcpError{class: "timeout", msg: fmt.Sprintf("mcp call %s: %v", toolName, ctx.Err())}
		}
		return nil, mcpError{class: "transport", msg: fmt.Sprintf("mcp call %s: %v", toolName, err)}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, mcpError{class: "transport", msg: fmt.Sprintf("read MCP response: %v", err)}
	}
	var rpc struct {
		JSONRPC string         `json:"jsonrpc"`
		ID      any            `json:"id"`
		Result  map[string]any `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    any    `json:"data,omitempty"`
		} `json:"error"`
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, mcpError{class: "http", msg: fmt.Sprintf("mcp HTTP status %d: %s", resp.StatusCode, trimMCPBody(bodyBytes))}
	}
	if err := json.Unmarshal(bodyBytes, &rpc); err != nil {
		return nil, mcpError{class: "decode", msg: fmt.Sprintf("decode MCP response: %v", err)}
	}
	if rpc.JSONRPC != "2.0" {
		return nil, mcpError{class: "decode", msg: "mcp response jsonrpc must be 2.0"}
	}
	if fmt.Sprint(rpc.ID) != fmt.Sprint(reqBody["id"]) {
		return nil, mcpError{class: "decode", msg: "mcp response id mismatch"}
	}
	if rpc.Error != nil {
		return nil, mcpError{class: "rpc", msg: fmt.Sprintf("mcp error %d: %s", rpc.Error.Code, rpc.Error.Message)}
	}
	if rpc.Result == nil {
		rpc.Result = map[string]any{}
	}
	return map[string]any{
		"serverURL": serverURL,
		"tool":      toolName,
		"result":    rpc.Result,
		"timeoutMs": timeoutMs,
	}, nil
}

type mcpError struct {
	class string
	msg   string
}

func (e mcpError) Error() string {
	return e.msg
}

func (e mcpError) FailureClass() string {
	return "mcp_" + e.class
}

func validateMCPServerURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return mcpError{class: "schema", msg: fmt.Sprintf("invalid serverURL: %v", err)}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return mcpError{class: "policy", msg: "serverURL scheme must be http or https"}
	}
	if u.Host == "" {
		return mcpError{class: "schema", msg: "serverURL host is required"}
	}
	if u.User != nil {
		return mcpError{class: "policy", msg: "serverURL must not include userinfo"}
	}
	return nil
}

func validateMCPTopLevelArgs(args map[string]any) error {
	allowed := map[string]struct{}{
		"serverURL": {}, "endpoint": {}, "url": {},
		"name": {}, "tool": {}, "toolName": {},
		"arguments": {}, "input": {}, "args": {},
		"timeoutMs": {}, "allowedArgs": {}, "allowedArgumentKeys": {},
	}
	for key := range args {
		if _, ok := allowed[key]; !ok {
			return mcpError{class: "schema", msg: fmt.Sprintf("unsupported mcp.call argument %q", key)}
		}
	}
	return nil
}

func validateMCPArgumentWhitelist(args, input map[string]any) error {
	allowed, ok := stringListArg(args, "allowedArgs", "allowedArgumentKeys")
	if !ok {
		return nil
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	for key := range input {
		if _, ok := allowedSet[key]; !ok {
			return mcpError{class: "policy", msg: fmt.Sprintf("mcp argument %q is not allowed", key)}
		}
	}
	return nil
}

func stringListArg(args map[string]any, keys ...string) ([]string, bool) {
	for _, key := range keys {
		v, ok := args[key]
		if !ok {
			continue
		}
		out := []string{}
		switch items := v.(type) {
		case []string:
			out = append(out, items...)
		case []any:
			for _, item := range items {
				if str, ok := item.(string); ok {
					out = append(out, str)
				}
			}
		case string:
			for _, item := range strings.Split(items, ",") {
				out = append(out, strings.TrimSpace(item))
			}
		default:
			return nil, true
		}
		clean := make([]string, 0, len(out))
		for _, item := range out {
			item = strings.TrimSpace(item)
			if item != "" {
				clean = append(clean, item)
			}
		}
		sort.Strings(clean)
		return clean, true
	}
	return nil, false
}

func trimMCPBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if len(text) <= 500 {
		return text
	}
	return text[:500] + "...<truncated>"
}

func firstStringArg(args map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := args[key].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func int64Arg(args map[string]any, key string, fallback int64) int64 {
	switch v := args[key].(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		n, err := v.Int64()
		if err == nil {
			return n
		}
	}
	return fallback
}
