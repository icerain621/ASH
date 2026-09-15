package session

import (
	"strings"
	"unicode/utf8"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/llmchat"
)

// AssistantReply is a persisted assistant prose reply for sessions without a bound run.
// Bound-run replies live on the run event ledger (assistant.delta / assistant.message).
type AssistantReply struct {
	TurnID    string   `json:"turnId"`
	Text      string   `json:"text"`
	Source    string   `json:"source,omitempty"` // "echo" | "llm" | adapter name
	Stopped   bool     `json:"stopped,omitempty"`
	Chunks    []string `json:"chunks,omitempty"`
	CreatedAt int64    `json:"createdAt,omitempty"`
}

// resolveAssistantText prefers a usable provider response string; otherwise returns an echo stub
// so blank sessions are not user-only.
func resolveAssistantText(prompt string, providerPayload map[string]any) (text, source string) {
	if msg, adapter := providerResponse(providerPayload); msg != "" {
		src := firstNonEmpty(adapter, "provider")
		return msg, src
	}
	return "已收到：" + prompt, "echo"
}

func providerResponse(payload map[string]any) (text, adapter string) {
	if payload == nil {
		return "", ""
	}
	adapter, _ = payload["providerAdapter"].(string)
	for _, key := range []string{"providerMessage", "acpMessage", "response", "text", "message"} {
		if v, ok := payload[key].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				return s, strings.TrimSpace(adapter)
			}
		}
	}
	return "", ""
}

// splitReplyChunks splits reply text into 2–4 chunks for assistant.delta streaming shape.
func splitReplyChunks(text string) []string {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return []string{""}
	}
	parts := 2
	if n > 40 {
		parts = 3
	}
	if n > 80 {
		parts = 4
	}
	if parts > n {
		parts = n
	}
	runes := []rune(text)
	out := make([]string, 0, parts)
	base := n / parts
	extra := n % parts
	off := 0
	for i := 0; i < parts; i++ {
		size := base
		if i < extra {
			size++
		}
		if size <= 0 {
			continue
		}
		out = append(out, string(runes[off:off+size]))
		off += size
	}
	if len(out) == 0 {
		return []string{text}
	}
	return out
}

// emitAssistantReply writes streaming assistant.delta + final assistant.message.
// With runId: Append to the run ledger. Without runId: store on view.Replies for synthesizeTurnEvents.
//
// Note: echo/provider replies complete synchronously today. Intent stop/cancel only cancels the bound
// run; stopped:true is reserved for a future mid-flight stream cancel.
func (s *Service) emitAssistantReply(view *View, turn Turn, text, source string) {
	s.emitAssistantReplyChunks(view, turn, text, source, splitReplyChunks(text))
}

func (s *Service) emitAssistantReplyChunks(view *View, turn Turn, text, source string, chunks []string) {
	if view == nil {
		return
	}
	if len(chunks) == 0 {
		chunks = []string{text}
	}
	stopped := false

	if strings.TrimSpace(view.RunID) != "" && s.events != nil {
		trace := firstNonEmpty(view.TraceID, view.RunID)
		for i, chunk := range chunks {
			_, _ = s.events.Append(view.RunID, trace, "assistant.delta", "info", map[string]any{
				"turnId": turn.ID, "text": chunk, "index": i,
			}, events.WithVisibility(events.VisibilityModelVisible))
		}
		_, _ = s.events.Append(view.RunID, trace, "assistant.message", "info", map[string]any{
			"turnId": turn.ID, "text": text, "stopped": stopped, "source": source,
		}, events.WithVisibility(events.VisibilityModelVisible))
		return
	}

	view.Replies = append(view.Replies, AssistantReply{
		TurnID: turn.ID, Text: text, Source: source, Stopped: stopped,
		Chunks: chunks, CreatedAt: turn.CreatedAt,
	})
}

// emitAssistantDelta appends one live delta to the run ledger (bound sessions only).
func (s *Service) emitAssistantDelta(view *View, turn Turn, text string, index int) {
	if view == nil || strings.TrimSpace(view.RunID) == "" || s.events == nil {
		return
	}
	trace := firstNonEmpty(view.TraceID, view.RunID)
	_, _ = s.events.Append(view.RunID, trace, "assistant.delta", "info", map[string]any{
		"turnId": turn.ID, "text": text, "index": index,
	}, events.WithVisibility(events.VisibilityModelVisible))
}

// emitAssistantMessageFinal writes the closing assistant.message (bound sessions only).
func (s *Service) emitAssistantMessageFinal(view *View, turn Turn, text, source string, stopped bool) {
	if view == nil || strings.TrimSpace(view.RunID) == "" || s.events == nil {
		return
	}
	trace := firstNonEmpty(view.TraceID, view.RunID)
	_, _ = s.events.Append(view.RunID, trace, "assistant.message", "info", map[string]any{
		"turnId": turn.ID, "text": text, "stopped": stopped, "source": source,
	}, events.WithVisibility(events.VisibilityModelVisible))
}

// buildChatMessages assembles recent turns (+ assistant replies when present) for LLM chat.
func buildChatMessages(view *View, newPrompt string) []llmchat.Message {
	const maxTurns = 12
	repliesByTurn := make(map[string]AssistantReply, len(view.Replies))
	for _, r := range view.Replies {
		repliesByTurn[r.TurnID] = r
	}
	turns := view.Turns
	// view.Turns already includes the new turn when called from PromptTurn; exclude it from history.
	hist := turns
	if len(hist) > 0 {
		hist = hist[:len(hist)-1]
	}
	if len(hist) > maxTurns {
		hist = hist[len(hist)-maxTurns:]
	}
	out := make([]llmchat.Message, 0, len(hist)*2+1)
	for _, t := range hist {
		out = append(out, llmchat.Message{Role: "user", Content: t.Prompt})
		if r, ok := repliesByTurn[t.ID]; ok && strings.TrimSpace(r.Text) != "" {
			out = append(out, llmchat.Message{Role: "assistant", Content: r.Text})
		}
	}
	out = append(out, llmchat.Message{Role: "user", Content: newPrompt})
	return out
}
