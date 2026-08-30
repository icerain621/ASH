package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// RPCRequest is one LF-delimited JSON line from stdin.
type RPCRequest struct {
	Type        string `json:"type"`
	SessionID   string `json:"sessionId,omitempty"`
	Goal        string `json:"goal,omitempty"`
	RunID       string `json:"runId,omitempty"`
	RepoRoot    string `json:"repoRoot,omitempty"`
	AutoApprove bool   `json:"autoApprove,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	SpaceID     string `json:"spaceId,omitempty"`
	ActorRole   string `json:"actorRole,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
}

// RPCEvent is one LF-delimited JSON response line.
type RPCEvent struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	SessionID string `json:"sessionId,omitempty"`
	Payload   any    `json:"payload,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ServeRPC reads LF-JSON requests and writes LF-JSON events until EOF.
func (s *Service) ServeRPC(r io.Reader, w io.Writer) error {
	sc := bufio.NewScanner(r)
	// Allow long prompts / event payloads.
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req RPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			_ = enc.Encode(RPCEvent{Type: "event", Name: "rpc.error", Error: "invalid json: " + err.Error()})
			continue
		}
		switch strings.TrimSpace(strings.ToLower(req.Type)) {
		case "session.start":
			view, err := s.Create(CreateRequest{
				Goal: req.Goal, RunID: req.RunID, RepoRoot: req.RepoRoot,
				SpaceID: req.SpaceID, ActorRole: req.ActorRole,
				CreatedBy: firstNonEmpty(req.CreatedBy, "rpc"), AutoApprove: req.AutoApprove,
			})
			if err != nil {
				_ = enc.Encode(RPCEvent{Type: "event", Name: "rpc.error", Error: err.Error()})
				continue
			}
			_ = enc.Encode(RPCEvent{Type: "event", Name: "session.started", SessionID: view.ID, Payload: view})
		case "turn.prompt":
			view, turn, err := s.PromptTurn(req.SessionID, TurnRequest{Prompt: req.Prompt})
			if err != nil {
				_ = enc.Encode(RPCEvent{
					Type: "event", Name: "rpc.error", SessionID: req.SessionID, Error: err.Error(),
				})
				continue
			}
			_ = enc.Encode(RPCEvent{
				Type: "event", Name: "turn.accepted", SessionID: view.ID,
				Payload: map[string]any{"turn": turn, "session": view},
			})
		default:
			_ = enc.Encode(RPCEvent{
				Type: "event", Name: "rpc.error", SessionID: req.SessionID,
				Error: fmt.Sprintf("unsupported type %q", req.Type),
			})
		}
	}
	return sc.Err()
}
