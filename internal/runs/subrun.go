package runs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/store"
)

// DefaultSubRunAllowlist is the safe tool set for spawned children unless narrowed further.
var DefaultSubRunAllowlist = []string{
	"read", "grep", "find", "ls", "git.status", "git.diff",
}

// SpawnRequest creates a child run under a parent.
type SpawnRequest struct {
	Scenario      ScenarioRef    `json:"scenario" binding:"required"`
	Inputs        map[string]any `json:"inputs" binding:"required"`
	PolicyProfile string         `json:"policyProfile"`
	AllowedTools  []string       `json:"allowedTools"`
	Reason        string         `json:"reason"`
	ActorRole     string         `json:"actorRole"`
}

// TreeNode is one node in a run spawn tree.
type TreeNode struct {
	Summary  Summary    `json:"summary"`
	Children []TreeNode `json:"children,omitempty"`
}

// TreeResponse is the spawn tree rooted at a run's root.
type TreeResponse struct {
	RootRunID string   `json:"rootRunId"`
	Tree      TreeNode `json:"tree"`
}

// Spawn creates and executes a sub-run under parentRunID.
func (s *Service) Spawn(parentRunID string, req SpawnRequest) (*CreateResponse, error) {
	parentRunID = strings.TrimSpace(parentRunID)
	if parentRunID == "" {
		return nil, fmt.Errorf("parent run id required")
	}
	var parent store.RunRecord
	if err := s.gdb().First(&parent, "id = ?", parentRunID).Error; err != nil {
		return nil, fmt.Errorf("parent run not found")
	}

	maxDepth := 2
	if s.harnessSvc != nil {
		if view, err := s.harnessSvc.LoadActive(firstNonEmpty(parent.SpaceID, "local"), "default"); err == nil && view != nil {
			if view.Spec.SubRun != nil && view.Spec.SubRun.MaxDepth > 0 {
				maxDepth = view.Spec.SubRun.MaxDepth
			}
		}
	}
	childDepth := parent.Depth + 1
	if childDepth > maxDepth {
		return nil, fmt.Errorf("sub-run depth %d exceeds maxDepth %d", childDepth, maxDepth)
	}

	allow := normalizeAllowlist(req.AllowedTools)
	if len(allow) == 0 {
		allow = append([]string(nil), DefaultSubRunAllowlist...)
	}
	if err := validateSubRunAllowlist(allow); err != nil {
		return nil, err
	}

	rootID := parent.RootRunID
	if rootID == "" {
		rootID = parent.ID
	}

	createReq := CreateRequest{
		Scenario:      req.Scenario,
		Inputs:        req.Inputs,
		PolicyProfile: firstNonEmpty(req.PolicyProfile, parent.PolicyProfile),
		SpaceID:       parent.SpaceID,
		ActorRole:     firstNonEmpty(req.ActorRole, parent.ActorRole),
	}
	if parent.RepoRoot != "" {
		createReq.Repo = &RepoRef{Root: parent.RepoRoot}
		if createReq.Inputs == nil {
			createReq.Inputs = map[string]any{}
		}
		if _, ok := createReq.Inputs["repoRoot"]; !ok {
			createReq.Inputs["repoRoot"] = parent.RepoRoot
		}
	}

	resp, err := s.createAndExecute(createReq, createOptions{
		parentRunID: parent.ID, rootRunID: rootID, depth: childDepth, toolAllowlist: allow,
	})
	if resp != nil {
		_, _ = s.eventsFor().Append(parent.ID, parent.TraceID, "run.spawned", "info", map[string]any{
			"childRunId": resp.RunID, "childTraceId": resp.TraceID,
			"depth": childDepth, "reason": strings.TrimSpace(req.Reason),
			"allowedTools": allow, "scenario": req.Scenario.Name,
		})
	}
	return resp, err
}

// Tree returns the spawn tree for the root of runID.
func (s *Service) Tree(runID string) (*TreeResponse, error) {
	var rec store.RunRecord
	if err := s.gdb().First(&rec, "id = ?", runID).Error; err != nil {
		return nil, err
	}
	rootID := rec.RootRunID
	if rootID == "" {
		rootID = rec.ID
	}
	var rows []store.RunRecord
	if err := s.gdb().Where("id = ? OR root_run_id = ?", rootID, rootID).
		Order("depth asc, started_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	byParent := map[string][]store.RunRecord{}
	var root *store.RunRecord
	for i := range rows {
		r := rows[i]
		if r.ID == rootID {
			cp := r
			root = &cp
			continue
		}
		byParent[r.ParentRunID] = append(byParent[r.ParentRunID], r)
	}
	if root == nil {
		return nil, fmt.Errorf("root run not found")
	}
	return &TreeResponse{
		RootRunID: rootID,
		Tree:      buildTreeNode(*root, byParent),
	}, nil
}

func buildTreeNode(rec store.RunRecord, byParent map[string][]store.RunRecord) TreeNode {
	node := TreeNode{Summary: *recordToSummary(rec)}
	for _, child := range byParent[rec.ID] {
		node.Children = append(node.Children, buildTreeNode(child, byParent))
	}
	return node
}

func normalizeAllowlist(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

var dangerousSubRunTools = map[string]struct{}{
	"write": {}, "edit": {}, "bash": {}, "runtime.command": {},
}

func validateSubRunAllowlist(tools []string) error {
	for _, t := range tools {
		if _, bad := dangerousSubRunTools[t]; bad {
			return fmt.Errorf("tool %q is not allowed on sub-runs without explicit parent policy (denied by default)", t)
		}
	}
	return nil
}

func parseAllowlistJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var tools []string
	if json.Unmarshal([]byte(raw), &tools) != nil {
		return nil
	}
	return tools
}

func allowlistPermits(allow []string, tool string) bool {
	if len(allow) == 0 {
		return true
	}
	tool = strings.TrimSpace(tool)
	for _, t := range allow {
		if t == tool {
			return true
		}
	}
	return false
}

// maxDepthFromHarness exposes harness default for tests.
func maxDepthFromSpec(spec *harness.SubRunSpec) int {
	if spec == nil || spec.MaxDepth <= 0 {
		return 2
	}
	return spec.MaxDepth
}
