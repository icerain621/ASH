package runs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ash-repwiki/ash/internal/artifacts"
)

const defaultContextTokenBudget = 32000

type compactionTracker struct {
	spillMax      int
	compactOn     bool
	triggerRatio  float64
	budgetTokens  int
	usedTokens    int
	compactedOnce bool
}

func (s *Service) newCompactionTracker(spaceID string) *compactionTracker {
	t := &compactionTracker{
		spillMax:     65536,
		compactOn:    true,
		triggerRatio: 0.85,
		budgetTokens: defaultContextTokenBudget,
	}
	if s == nil || s.harnessSvc == nil {
		return t
	}
	view, err := s.harnessSvc.LoadActive(firstNonEmpty(spaceID, "local"), "default")
	if err != nil || view == nil {
		return t
	}
	if view.Spec.Sandbox.SpillMaxBytes > 0 {
		t.spillMax = view.Spec.Sandbox.SpillMaxBytes
	}
	if view.Spec.Compaction != nil {
		t.compactOn = view.Spec.Compaction.Enabled
		if view.Spec.Compaction.TriggerTokenRatio > 0 {
			t.triggerRatio = view.Spec.Compaction.TriggerTokenRatio
		}
	}
	return t
}

func estimateTokens(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	return (len(b) + 3) / 4
}

func truncatePreview(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// maybeSpillToolOutput writes oversized tool output to run artifacts and returns a compact payload.
func (s *Service) maybeSpillToolOutput(runID, traceID, stepID, tool string, output map[string]any, spillMax int) map[string]any {
	if output == nil || spillMax <= 0 || s == nil || s.db == nil {
		return output
	}
	raw, err := json.Marshal(output)
	if err != nil || len(raw) <= spillMax {
		return output
	}
	runDir := s.db.RunDir(runID)
	_ = artifacts.EnsureRunLayout(runDir)
	safeStep := strings.ReplaceAll(stepID, "/", "_")
	safeTool := strings.ReplaceAll(tool, "/", "_")
	name := fmt.Sprintf("spill_%s_%s.json", safeStep, safeTool)
	rel := filepath.ToSlash(filepath.Join("artifacts", name))
	path := filepath.Join(runDir, "artifacts", name)
	if werr := os.WriteFile(path, raw, artifacts.DefaultFilePerm); werr != nil {
		_, _ = s.eventsFor().Append(runID, traceID, "tool.spill_failed", "warn", map[string]any{
			"stepId": stepID, "tool": tool, "error": werr.Error(), "bytes": len(raw),
		})
		return output
	}
	_, _ = s.eventsFor().Append(runID, traceID, "tool.spilled", "info", map[string]any{
		"stepId": stepID, "tool": tool, "spilledPath": rel, "spilledBytes": len(raw),
	})
	return map[string]any{
		"spilled":      true,
		"spilledPath":  rel,
		"spilledBytes": len(raw),
		"preview":      truncatePreview(string(raw), 512),
	}
}

func (s *Service) maybeCompact(runID, traceID, stepID string, t *compactionTracker, sample []byte) {
	if s == nil || t == nil || !t.compactOn || t.compactedOnce || t.budgetTokens <= 0 {
		return
	}
	t.usedTokens += estimateTokens(sample)
	ratio := float64(t.usedTokens) / float64(t.budgetTokens)
	if ratio < t.triggerRatio {
		return
	}
	t.compactedOnce = true
	_, _ = s.eventsFor().Append(runID, traceID, "harness.compaction", "info", map[string]any{
		"stepId":           stepID,
		"triggerTokenRatio": t.triggerRatio,
		"estimatedTokens":  t.usedTokens,
		"budgetTokens":     t.budgetTokens,
		"ratio":            ratio,
		"summary":          "lossy compaction stub: prior tool outputs spilled or truncated for context budget",
	})
}
