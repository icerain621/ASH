package session

import (
	"strings"
	"unicode/utf8"

	"github.com/ash-repwiki/ash/internal/events"
)

// AssistantReply is a persisted assistant prose reply for sessions without a bound run.
// Bound-run replies live on the run event ledger (assistant.delta / assistant.message).
type AssistantReply struct {
	TurnID    string   `json:"turnId"`
	Text      string   `json:"text"`
	Source    string   `json:"source,omitempty"` // "echo" | "acp"
	Stopped   bool     `json:"stopped,omitempty"`
	Chunks    []string `json:"chunks,omitempty"`
	CreatedAt int64    `json:"createdAt,omitempty"`
}

// resolveAssistantText prefers a usable ACP response string; otherwise returns an echo stub
// so blank sessions are not user-only.
func resolveAssistantText(prompt string, acpPayload map[string]any) (text, source string) {
	if msg := acpResponseString(acpPayload); msg != "" {
		return msg, "acp"
	}
	return "已收到：" + prompt, "echo"
}

func acpResponseString(acpPayload map[string]any) string {
	if acpPayload == nil {
		return ""
	}
	for _, key := range []string{"acpMessage", "response", "text", "message"} {
		if v, ok := acpPayload[key].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		}
	}
	return ""
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
// Note: echo/ACP replies complete synchronously today. Intent stop/cancel only cancels the bound
// run; stopped:true is reserved for a future mid-flight stream.
func (s *Service) emitAssistantReply(view *View, turn Turn, text, source string) {
	if view == nil {
		return
	}
	chunks := splitReplyChunks(text)
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
