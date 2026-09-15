package session

import (
	"context"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/llmchat"
	"github.com/ash-repwiki/ash/internal/skills"
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

// replyViaSkillLLM runs the skill body as system prompt (+ args as user) when ASH_LLM is configured.
func (s *Service) replyViaSkillLLM(view *View, turn Turn, sk *skills.Skill, args string) bool {
	if view == nil || sk == nil || !llmchat.Configured() {
		return false
	}
	system := strings.TrimSpace(sk.Body)
	if system == "" {
		system = strings.TrimSpace(sk.Description)
	}
	if system == "" {
		return false
	}
	userMsg := strings.TrimSpace(args)
	if userMsg == "" {
		userMsg = "请根据技能说明开始协助。"
	}
	client := llmchat.NewFromEnv()
	messages := []llmchat.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: userMsg},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	bound := strings.TrimSpace(view.RunID) != "" && s.events != nil
	var chunks []string
	index := 0
	full, err := client.Stream(ctx, messages, func(delta string) {
		if delta == "" {
			return
		}
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
		s.emitAssistantMessageFinal(view, turn, full, "skill", false)
		return true
	}
	if len(chunks) == 0 {
		chunks = splitReplyChunks(full)
	}
	s.emitAssistantReplyChunks(view, turn, full, "skill", chunks)
	return true
}
