package interaction

import (
	"strings"

	"github.com/google/uuid"
)

const (
	MetaThreadID   = "threadId"
	MetaThreadKind = "threadKind"
	MetaThreadRun  = "threadRunId"
	ThreadKindMain = "main"
)

// EnsureMainThread binds a stable main thread id onto session meta (GV01 minimal).
// Full Thread table / Fold arrives in later sprints; meta is the bridge for P0.
func EnsureMainThread(meta map[string]any, sessionID, runID string) (threadID string, out map[string]any, created bool) {
	out = meta
	if out == nil {
		out = map[string]any{}
	}
	if existing, ok := out[MetaThreadID].(string); ok && strings.TrimSpace(existing) != "" {
		return strings.TrimSpace(existing), out, false
	}
	_ = sessionID
	threadID = "th_" + uuid.NewString()
	out[MetaThreadID] = threadID
	out[MetaThreadKind] = ThreadKindMain
	if run := strings.TrimSpace(runID); run != "" {
		out[MetaThreadRun] = run
	}
	return threadID, out, true
}
