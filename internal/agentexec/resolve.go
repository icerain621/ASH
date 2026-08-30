package agentexec

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ProbeReport is a lightweight ExecGo / provider readiness snapshot.
type ProbeReport struct {
	Adapter   string `json:"adapter"`
	Kind      string `json:"kind"`
	OK        bool   `json:"ok"`
	Message   string `json:"message,omitempty"`
	CheckedAt int64  `json:"checkedAt"`
}

// Resolve picks an executor for a harness provider kind.
func Resolve(kind string) Executor {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "execgo", "execgo_codex":
		return NewExecGoCodexExecutor()
	case "static":
		return StaticExecutor{}
	case "acp_sdk":
		return NewACPExecutor()
	default:
		return StaticExecutor{}
	}
}

// AdapterNameOf returns AdapterName when available.
func AdapterNameOf(exec Executor) string {
	if named, ok := exec.(interface{ AdapterName() string }); ok {
		return named.AdapterName()
	}
	return "unknown"
}

// ProbeExecGo runs health checks against the ExecGo bridge (no Codex task).
func ProbeExecGo(ctx context.Context) ProbeReport {
	now := time.Now().UTC().Unix()
	e := NewExecGoCodexExecutor()
	if err := e.health(ctx); err != nil {
		return ProbeReport{
			Adapter: e.AdapterName(), Kind: "execgo", OK: false,
			Message: err.Error(), CheckedAt: now,
		}
	}
	return ProbeReport{
		Adapter: e.AdapterName(), Kind: "execgo", OK: true,
		Message: "execgocli health+tools ok", CheckedAt: now,
	}
}

// NormalizeProviderKind maps harness provider.kind to resolve key.
func NormalizeProviderKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	switch k {
	case "", "execgo_codex":
		return "execgo"
	case "acp", "acp-sdk":
		return "acp_sdk"
	default:
		return k
	}
}

// DescribeKind returns a short human label.
func DescribeKind(kind string) string {
	switch NormalizeProviderKind(kind) {
	case "static":
		return "static"
	case "execgo":
		return "execgo"
	case "acp_sdk":
		return "acp_sdk"
	default:
		return fmt.Sprintf("%s (fallback static)", kind)
	}
}
