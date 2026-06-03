package modelrouter

import (
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/store"
)

type Provider struct {
	ID                string `json:"id"`
	Provider          string `json:"provider"`
	Model             string `json:"model"`
	Role              string `json:"role"`
	Status            string `json:"status"`
	BaseURLConfigured bool   `json:"baseUrlConfigured"`
	APIKeyConfigured  bool   `json:"apiKeyConfigured"`
	InputMicrosPer1K  int64  `json:"inputMicrosPer1K"`
	OutputMicrosPer1K int64  `json:"outputMicrosPer1K"`
}

type Request struct {
	RunID        string `json:"runId,omitempty"`
	StepID       string `json:"stepId,omitempty"`
	UseCase      string `json:"useCase,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
	InputTokens  int64  `json:"inputTokens,omitempty"`
	OutputTokens int64  `json:"outputTokens,omitempty"`
}

type Decision struct {
	Provider     Provider `json:"provider"`
	Status       string   `json:"status"`
	Reason       string   `json:"reason,omitempty"`
	FallbackUsed bool     `json:"fallbackUsed"`
	InputTokens  int64    `json:"inputTokens"`
	OutputTokens int64    `json:"outputTokens"`
	CostMicros   int64    `json:"costMicros"`
}

type Router struct {
	providers []Provider
}

func New(providers []Provider) Router {
	out := make([]Provider, len(providers))
	copy(out, providers)
	return Router{providers: out}
}

func NewFromEnv() Router {
	return Router{providers: []Provider{
		providerFromEnv("ASH_MODEL_PRIMARY", "primary", "default"),
		providerFromEnv("ASH_MODEL_FALLBACK", "fallback", "fallback"),
	}}
}

func (r Router) Providers() []Provider {
	out := make([]Provider, len(r.providers))
	copy(out, r.providers)
	return out
}

func (r Router) Route(req Request) Decision {
	req.InputTokens = normalizeTokens(req.InputTokens, req.Prompt)
	if req.OutputTokens < 0 {
		req.OutputTokens = 0
	}
	var configured []Provider
	for _, p := range r.providers {
		if providerAvailable(p) {
			configured = append(configured, p)
		}
	}
	if len(configured) == 0 {
		p := Provider{ID: "none", Provider: "none", Model: "none", Role: "none", Status: "not_configured"}
		if len(r.providers) > 0 {
			p = r.providers[0]
		}
		return Decision{
			Provider: p, Status: "not_configured",
			Reason:       "no model provider is configured or available",
			InputTokens:  req.InputTokens,
			OutputTokens: req.OutputTokens,
			CostMicros:   estimateCostMicros(p, req.InputTokens, req.OutputTokens),
		}
	}
	selected := configured[0]
	fallback := selected.Role == "fallback"
	return Decision{
		Provider: selected, Status: "routed", FallbackUsed: fallback,
		Reason:       routeReason(selected, fallback),
		InputTokens:  req.InputTokens,
		OutputTokens: req.OutputTokens,
		CostMicros:   estimateCostMicros(selected, req.InputTokens, req.OutputTokens),
	}
}

func UsageRow(dec Decision, req Request) store.ModelUsage {
	return store.ModelUsage{
		ID:    "model_usage_" + uuid.NewString(),
		RunID: req.RunID, StepID: req.StepID,
		Provider: dec.Provider.ID, Model: dec.Provider.Model,
		InputTokens: dec.InputTokens, OutputTokens: dec.OutputTokens,
		CostMicros: dec.CostMicros, Status: dec.Status,
		CreatedAt: time.Now().UTC(),
	}
}

func providerFromEnv(prefix, id, role string) Provider {
	provider := envOr(prefix+"_PROVIDER", "openai-compatible")
	model := envOr(prefix+"_MODEL", "not-configured")
	baseURL := os.Getenv(prefix + "_BASE_URL")
	apiKey := os.Getenv(prefix + "_API_KEY")
	status := os.Getenv(prefix + "_STATUS")
	if status == "" {
		if baseURL != "" || apiKey != "" {
			status = "available"
		} else {
			status = "not_configured"
		}
	}
	return Provider{
		ID: id, Provider: provider, Model: model, Role: role, Status: normalizeStatus(status),
		BaseURLConfigured: baseURL != "", APIKeyConfigured: apiKey != "",
		InputMicrosPer1K:  envInt64(prefix+"_INPUT_MICROS_PER_1K", 0),
		OutputMicrosPer1K: envInt64(prefix+"_OUTPUT_MICROS_PER_1K", 0),
	}
}

func providerAvailable(p Provider) bool {
	switch normalizeStatus(p.Status) {
	case "available", "configured":
		return true
	default:
		return false
	}
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	status = strings.ReplaceAll(status, "-", "_")
	if status == "" {
		return "not_configured"
	}
	return status
}

func normalizeTokens(tokens int64, prompt string) int64 {
	if tokens > 0 {
		return tokens
	}
	if prompt == "" {
		return 0
	}
	runes := utf8.RuneCountInString(prompt)
	if runes == 0 {
		return 0
	}
	return int64((runes + 3) / 4)
}

func estimateCostMicros(p Provider, inputTokens, outputTokens int64) int64 {
	return (inputTokens*p.InputMicrosPer1K + outputTokens*p.OutputMicrosPer1K) / 1000
}

func routeReason(p Provider, fallback bool) string {
	if fallback {
		return "primary provider unavailable; fallback selected"
	}
	return "primary provider selected"
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}
