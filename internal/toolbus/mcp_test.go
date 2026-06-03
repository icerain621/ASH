package toolbus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPCallSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["method"] != "tools/call" {
			t.Fatalf("method=%v", req["method"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result":  map[string]any{"content": []any{map[string]any{"type": "text", "text": "ok"}}},
		})
	}))
	defer srv.Close()

	res := DefaultBus().Call(Context{}, CallRequest{
		Tool: "mcp.call",
		Args: map[string]any{
			"serverURL":   srv.URL,
			"tool":        "demo.echo",
			"arguments":   map[string]any{"message": "hello"},
			"allowedArgs": []any{"message"},
		},
	})
	if !res.OK {
		t.Fatalf("expected ok: %+v", res)
	}
	if res.Output["tool"] != "demo.echo" {
		t.Fatalf("tool=%v", res.Output["tool"])
	}
}

func TestMCPCallError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"error":   map[string]any{"code": -32000, "message": "boom"},
		})
	}))
	defer srv.Close()

	res := DefaultBus().Call(Context{}, CallRequest{
		Tool: "mcp.call",
		Args: map[string]any{"serverURL": srv.URL, "tool": "demo.fail"},
	})
	if res.OK {
		t.Fatalf("expected failure: %+v", res)
	}
	if res.FailureClass != "mcp_rpc" {
		t.Fatalf("failureClass=%q want mcp_rpc", res.FailureClass)
	}
}

func TestMCPCallRejectsUnknownTopLevelArg(t *testing.T) {
	res := DefaultBus().Call(Context{}, CallRequest{
		Tool: "mcp.call",
		Args: map[string]any{
			"serverURL": "http://127.0.0.1:1",
			"tool":      "demo.echo",
			"headers":   map[string]any{"Authorization": "secret"},
		},
	})
	if res.OK {
		t.Fatalf("expected failure: %+v", res)
	}
	if res.FailureClass != "mcp_schema" {
		t.Fatalf("failureClass=%q want mcp_schema", res.FailureClass)
	}
}

func TestMCPCallRejectsArgumentsOutsideWhitelist(t *testing.T) {
	res := DefaultBus().Call(Context{}, CallRequest{
		Tool: "mcp.call",
		Args: map[string]any{
			"serverURL":   "http://127.0.0.1:1",
			"tool":        "demo.echo",
			"arguments":   map[string]any{"message": "hello", "secret": "nope"},
			"allowedArgs": []any{"message"},
		},
	})
	if res.OK {
		t.Fatalf("expected failure: %+v", res)
	}
	if res.FailureClass != "mcp_policy" {
		t.Fatalf("failureClass=%q want mcp_policy", res.FailureClass)
	}
}

func TestMCPCallRejectsExcessiveTimeout(t *testing.T) {
	res := DefaultBus().Call(Context{}, CallRequest{
		Tool: "mcp.call",
		Args: map[string]any{
			"serverURL": "http://127.0.0.1:1",
			"tool":      "demo.echo",
			"timeoutMs": 120001,
		},
	})
	if res.OK {
		t.Fatalf("expected failure: %+v", res)
	}
	if res.FailureClass != "mcp_policy" {
		t.Fatalf("failureClass=%q want mcp_policy", res.FailureClass)
	}
}

func TestMCPCallRejectsUnsafeServerURL(t *testing.T) {
	cases := []struct {
		name  string
		url   string
		class string
	}{
		{name: "scheme", url: "file:///tmp/mcp.sock", class: "mcp_policy"},
		{name: "userinfo", url: "https://user:pass@example.com/mcp", class: "mcp_policy"},
		{name: "host", url: "https:///mcp", class: "mcp_schema"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := DefaultBus().Call(Context{}, CallRequest{
				Tool: "mcp.call",
				Args: map[string]any{"serverURL": tc.url, "tool": "demo.echo"},
			})
			if res.OK {
				t.Fatalf("expected failure: %+v", res)
			}
			if res.FailureClass != tc.class {
				t.Fatalf("failureClass=%q want %s", res.FailureClass, tc.class)
			}
		})
	}
}

func TestMCPCallRejectsMismatchedJSONRPCEnvelope(t *testing.T) {
	tests := []struct {
		name string
		resp map[string]any
	}{
		{name: "jsonrpc", resp: map[string]any{"jsonrpc": "1.0", "id": "ash-mcp-demo.echo", "result": map[string]any{}}},
		{name: "id", resp: map[string]any{"jsonrpc": "2.0", "id": "wrong", "result": map[string]any{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(tt.resp)
			}))
			defer srv.Close()
			res := DefaultBus().Call(Context{}, CallRequest{
				Tool: "mcp.call",
				Args: map[string]any{"serverURL": srv.URL, "tool": "demo.echo"},
			})
			if res.OK {
				t.Fatalf("expected failure: %+v", res)
			}
			if res.FailureClass != "mcp_decode" {
				t.Fatalf("failureClass=%q want mcp_decode", res.FailureClass)
			}
		})
	}
}
