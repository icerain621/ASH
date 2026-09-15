package session

import (
	"context"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/llmchat"
)

// replyViaLLM streams (or completes) an OpenAI-compatible chat reply.
// emitTurn is invoked once before the first delta (bound run) so session.turn precedes assistant.*.
// Returns false when the LLM call fails so the caller can fall through to provider/echo.
func (s *Service) replyViaLLM(view *View, turn Turn, prompt string, emitTurn func(map[string]any)) bool {
	if view == nil || !llmchat.Configured() {
		return false
	}
	client := llmchat.NewFromEnv()
	messages := buildChatMessages(view, prompt)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	bound := strings.TrimSpace(view.RunID) != "" && s.events != nil
	turned := false
	ensureTurn := func() {
		if turned || !bound || emitTurn == nil {
			return
		}
		emitTurn(map[string]any{"llm": true, "llmModel": client.Model()})
		turned = true
	}

	var chunks []string
	index := 0
	full, err := client.Stream(ctx, messages, func(delta string) {
		if delta == "" {
			return
		}
		ensureTurn()
		chunks = append(chunks, delta)
		if bound {
			s.emitAssistantDelta(view, turn, delta, index)
			index++
		}
	})
	if err != nil || strings.TrimSpace(full) == "" {
		return false
	}

	if bound {
		ensureTurn()
		s.emitAssistantMessageFinal(view, turn, full, "llm", false)
		return true
	}

	if len(chunks) == 0 {
		chunks = splitReplyChunks(full)
	}
	s.emitAssistantReplyChunks(view, turn, full, "llm", chunks)
	return true
}
