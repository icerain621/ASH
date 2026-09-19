package runs

import (
	"encoding/json"
	"strings"

	"github.com/ash-repwiki/ash/internal/spacepolicy"
	"github.com/ash-repwiki/ash/internal/store"
)

// stripClientApprovalInputs removes reserved underscore approval keys from client
// Create/Replay inputs so callers cannot plant allow-lists or step approvals.
func stripClientApprovalInputs(inputs map[string]any) {
	if inputs == nil {
		return
	}
	for k := range inputs {
		if strings.HasPrefix(k, "_allowed") || strings.HasPrefix(k, "_approved") {
			delete(inputs, k)
		}
	}
}

func normalizeApproveScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case ApproveScopeSession, "allow_session":
		return ApproveScopeSession
	default:
		// Fail-closed: unknown / empty → once (never YOLO / never widen).
		return ApproveScopeOnce
	}
}

// resolveApproveTool forces the tool from pending gate evidence. Client-supplied
// tool is accepted only when it matches evidence (or evidence is empty and tool is empty).
func resolveApproveTool(evidenceTool, clientTool string) (string, error) {
	evidenceTool = strings.TrimSpace(evidenceTool)
	clientTool = strings.TrimSpace(clientTool)
	if clientTool != "" && evidenceTool != "" && clientTool != evidenceTool {
		return "", ErrApproveToolMismatch
	}
	if clientTool != "" && evidenceTool == "" {
		return "", ErrApproveToolMismatch
	}
	return evidenceTool, nil
}

func pendingApprovalTool(s *Service, runID, stepID string) string {
	if s == nil || runID == "" {
		return ""
	}
	var row store.ApprovalRequest
	q := s.gdb().Where("run_id = ?", runID)
	if stepID != "" {
		q = q.Where("step_id = ?", stepID)
	}
	if err := q.Order("updated_at desc").First(&row).Error; err != nil {
		return ""
	}
	if strings.TrimSpace(row.EvidenceJSON) == "" || row.EvidenceJSON == "{}" {
		return ""
	}
	var ev struct {
		Tool string `json:"tool"`
	}
	if err := json.Unmarshal([]byte(row.EvidenceJSON), &ev); err != nil {
		return ""
	}
	return strings.TrimSpace(ev.Tool)
}

func appendAllowedToolSession(inputs map[string]any, tool string) {
	tool = strings.TrimSpace(tool)
	if inputs == nil || tool == "" {
		return
	}
	list := approvedStepList(inputs["_allowedToolsSession"])
	for _, item := range list {
		if item == tool {
			inputs["_allowedToolsSession"] = list
			return
		}
	}
	inputs["_allowedToolsSession"] = append(list, tool)
}

func toolInAllowedSessionList(v any, tool string) bool {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return false
	}
	for _, item := range approvedStepList(v) {
		if item == tool {
			return true
		}
	}
	return false
}

func spaceToolPresetAllows(s *Service, spaceID, tool string) bool {
	if s == nil || strings.TrimSpace(tool) == "" {
		return false
	}
	spaceID = firstNonEmpty(strings.TrimSpace(spaceID), "local")
	var row store.SpacePolicyPack
	if err := s.gdb().First(&row, "space_id = ?", spaceID).Error; err != nil {
		return false
	}
	return spacepolicy.PresetAllowsTool(row.BodyJSON, tool)
}
