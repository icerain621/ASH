package lenses

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/observability"
	"github.com/ash-repwiki/ash/internal/store"
)

// LensEvent is one admitted spine event for a product lens.
type LensEvent struct {
	Type     string         `json:"type"`
	Lens     Lens           `json:"lens"`
	RunID    string         `json:"runId,omitempty"`
	Ts       int64          `json:"ts"`
	Payload  map[string]any `json:"payload,omitempty"`
	Redacted bool           `json:"redacted,omitempty"`
}

// CountItem is a named counter for global chips.
type CountItem struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}

// GlobalView is the global product lens (I02).
type GlobalView struct {
	GeneratedAt    int64       `json:"generatedAt"`
	RunningRuns    int         `json:"runningRuns"`
	FailedClusters []CountItem `json:"failedClusters"`
	TemplateUsage  []CountItem `json:"templateUsage"`
	GateRejects    int         `json:"gateRejects"`
	ReviewPending  int         `json:"reviewPending"`
	DoctorChip     string      `json:"doctorChip"`
}

// AgentView is the agent waterfall lens (I03).
type AgentView struct {
	RunID     string                   `json:"runId"`
	TraceID   string                   `json:"traceId"`
	Events    []LensEvent              `json:"events"`
	Waterfall *observability.Waterfall `json:"waterfall,omitempty"`
}

// MemoryStage is one step on the memory lineage.
type MemoryStage struct {
	Stage   string `json:"stage"`
	At      int64  `json:"at,omitempty"`
	Detail  string `json:"detail,omitempty"`
	EventID string `json:"eventId,omitempty"`
}

// MemoryLineageView is the memory lens (I04).
type MemoryLineageView struct {
	MemoryID string        `json:"memoryId"`
	Status   string        `json:"status"`
	Title    string        `json:"title"`
	Stages   []MemoryStage `json:"stages"`
	Events   []LensEvent   `json:"events"`
	Redacted bool          `json:"redacted"`
}

// BuildGlobal aggregates space-scoped counters from runs, events, and memory candidates.
func BuildGlobal(db *store.DB, spaceID string) (GlobalView, error) {
	space := strings.TrimSpace(spaceID)
	if space == "" {
		space = "local"
	}
	out := GlobalView{GeneratedAt: time.Now().UTC().UnixMilli(), DoctorChip: "BOOT"}
	var running int64
	if err := db.Model(&store.RunRecord{}).
		Where("space_id = ? AND status IN ?", space, []string{"running", "waiting_approval"}).
		Count(&running).Error; err != nil {
		return GlobalView{}, err
	}
	out.RunningRuns = int(running)

	var failed []store.RunRecord
	if err := db.Where("space_id = ? AND status IN ?", space, []string{"failed", "canceled"}).
		Order("updated_at desc").Limit(200).Find(&failed).Error; err != nil {
		return GlobalView{}, err
	}
	clusters := map[string]int{}
	for _, run := range failed {
		code := strings.TrimSpace(run.ErrorCode)
		if code == "" {
			code = run.Status
		}
		clusters[code]++
	}
	out.FailedClusters = sortedCounts(clusters)

	var candidates int64
	if err := db.Model(&store.MemoryRecord{}).
		Where("space_id = ? AND status = ?", space, "candidate").
		Count(&candidates).Error; err != nil {
		return GlobalView{}, err
	}
	out.ReviewPending = int(candidates)

	var runIDs []string
	if err := db.Model(&store.RunRecord{}).Where("space_id = ?", space).
		Order("updated_at desc").Limit(100).Pluck("id", &runIDs).Error; err != nil {
		return GlobalView{}, err
	}
	if len(runIDs) > 0 {
		var events []store.RunEvent
		if err := db.Where("run_id IN ?", runIDs).Order("ts desc").Limit(500).Find(&events).Error; err != nil {
			return GlobalView{}, err
		}
		templates := map[string]int{}
		for _, ev := range events {
			switch ev.Type {
			case "template.finished":
				payload := decodePayload(ev.PayloadJSON)
				if id, _ := payload["templateId"].(string); strings.TrimSpace(id) != "" {
					templates[id]++
				}
			case "scenario.step_deprecated", "agent.gate_rejected", "context.pack_rejected":
				out.GateRejects++
			}
		}
		out.TemplateUsage = sortedCounts(templates)
	}
	return out, nil
}

// BuildAgent returns agent-lens events plus the run waterfall.
func BuildAgent(db *store.DB, runID string) (AgentView, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return AgentView{}, fmt.Errorf("runId is required")
	}
	wf, err := observability.BuildWaterfall(db, runID)
	if err != nil {
		return AgentView{}, err
	}
	var events []store.RunEvent
	if err := db.Where("run_id = ?", runID).Order("ts asc").Limit(500).Find(&events).Error; err != nil {
		return AgentView{}, err
	}
	out := AgentView{RunID: runID, TraceID: wf.TraceID, Waterfall: wf}
	for _, ev := range events {
		if !Admit(ev.Type, LensAgent) {
			continue
		}
		out.Events = append(out.Events, LensEvent{
			Type: ev.Type, Lens: LensAgent, RunID: ev.RunID,
			Ts: ev.TS, Payload: decodePayload(ev.PayloadJSON),
		})
	}
	return out, nil
}

// BuildMemoryLineage walks candidate → review → approved → inject → hit_used → consolidate/forget.
func BuildMemoryLineage(db *store.DB, memoryID string) (MemoryLineageView, error) {
	memoryID = strings.TrimSpace(memoryID)
	if memoryID == "" {
		return MemoryLineageView{}, fmt.Errorf("memory id is required")
	}
	var row store.MemoryRecord
	if err := db.First(&row, "id = ?", memoryID).Error; err != nil {
		return MemoryLineageView{}, err
	}
	out := MemoryLineageView{
		MemoryID: row.ID,
		Status:   row.Status,
		Title:    redactTitle(row.Title),
		Redacted: true,
	}
	out.Stages = append(out.Stages, MemoryStage{Stage: "candidate", At: row.CreatedAt.UTC().UnixMilli(), Detail: "created"})
	if row.Status == "approved" || row.Status == "active" {
		out.Stages = append(out.Stages, MemoryStage{Stage: "approved", At: row.UpdatedAt.UTC().UnixMilli()})
	}
	var events []store.RunEvent
	_ = db.Where("payload_json LIKE ?", "%"+memoryID+"%").Order("ts asc").Limit(200).Find(&events)
	for _, ev := range events {
		if !Admit(ev.Type, LensMemory) {
			continue
		}
		payload := decodePayload(ev.PayloadJSON)
		redactPayload(payload)
		out.Events = append(out.Events, LensEvent{
			Type: ev.Type, Lens: LensMemory, RunID: ev.RunID,
			Ts: ev.TS, Payload: payload, Redacted: true,
		})
		stage := memoryStageFor(ev.Type)
		if stage != "" {
			out.Stages = append(out.Stages, MemoryStage{
				Stage: stage, At: ev.TS, EventID: ev.ID, Detail: ev.Type,
			})
		}
	}
	return out, nil
}

func memoryStageFor(eventType string) string {
	switch eventType {
	case "memory.injected":
		return "injected"
	case "memory.hit_used":
		return "hit_used"
	case "memory.consolidate_proposed":
		return "consolidate_proposed"
	case "memory.forget_proposed":
		return "forget_proposed"
	case "memory.feedback":
		return "feedback"
	default:
		return ""
	}
}

func decodePayload(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	return m
}

func redactPayload(m map[string]any) {
	if m == nil {
		return
	}
	for _, key := range []string{"body", "content", "text", "secret", "raw"} {
		if _, ok := m[key]; ok {
			m[key] = "[redacted]"
		}
	}
}

func redactTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "(untitled)"
	}
	return title
}

func sortedCounts(m map[string]int) []CountItem {
	out := make([]CountItem, 0, len(m))
	for id, n := range m {
		out = append(out, CountItem{ID: id, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].ID < out[j].ID
		}
		return out[i].Count > out[j].Count
	})
	return out
}
