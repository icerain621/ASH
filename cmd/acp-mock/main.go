package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ash-repwiki/ash/internal/agentexec"
)

// Minimal ACP control-plane mock for local smoke (Sprint DX4).
func main() {
	addr := flag.String("addr", "127.0.0.1:0", "listen address (host:port; port 0 = ephemeral)")
	flag.Parse()

	var seq atomic.Int64
	var mu sync.Mutex
	tasks := map[string]agentexec.ACPTaskResultV1{}

	mux := http.NewServeMux()
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"service":"acp-mock"}`))
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var task agentexec.ACPTaskV1
		if err := json.Unmarshal(body, &task); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := task.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id := fmt.Sprintf("mock-%d", seq.Add(1))
		res := agentexec.ACPTaskResultV1{
			Schema: agentexec.ACPTaskSchemaV1, OK: true, TaskID: id, Status: "success",
			Message: "acp-mock accepted",
			Output:  map[string]any{"echoPrompt": task.Prompt, "sessionId": task.SessionID},
		}
		mu.Lock()
		tasks[id] = res
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})
	mux.HandleFunc("/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/tasks/")
		parts := strings.Split(path, "/")
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		id := parts[0]
		if len(parts) == 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mu.Lock()
		res, ok := tasks[id]
		mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	base := "http://" + ln.Addr().String()
	fmt.Fprintln(os.Stdout, base)
	log.Printf("acp-mock listening on %s", base)
	if err := http.Serve(ln, mux); err != nil {
		log.Fatal(err)
	}
}
