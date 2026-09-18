package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/skills"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

// Command source labels for GET /agents/commands.
const (
	CommandSourceBuiltin = "builtin"
	CommandSourceSkill   = "skill"
	CommandSourceMCP     = "mcp"
)

// PermissionMode values for session seats (P5).
const (
	PermissionReadOnly       = "read-only"
	PermissionWorkspaceWrite = "workspace-write"
	PermissionFull           = "full"
)

// AgentMode values for session UX (coding vs general assistant).
const (
	AgentModeCoding  = "coding"
	AgentModeGeneral = "general"
)

// CommandItem is one slash/skill/mcp command catalog entry.
type CommandItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"` // builtin | skill | mcp
}

// CommandsResponse is returned by GET /agents/commands.
type CommandsResponse struct {
	Items []CommandItem `json:"items"`
}

// ModelItem is one provider seat option.
type ModelItem struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	ProviderKind string `json:"providerKind"`
}

// ModelsResponse is returned by GET /agents/models.
type ModelsResponse struct {
	Items []ModelItem `json:"items"`
}

// BuiltinCommands returns the always-available slash commands.
func BuiltinCommands() []CommandItem {
	return []CommandItem{
		{Name: "/help", Description: "列出可用命令", Source: CommandSourceBuiltin},
		{Name: "/clear", Description: "清空当前会话 transcript", Source: CommandSourceBuiltin},
	}
}

// ListCommands builds the command catalog (builtin + best-effort skills).
func ListCommands(repoRoot string) CommandsResponse {
	return ListCommandsForSpace(nil, "", repoRoot)
}

// ListCommandsForSpace builds builtin + skills + active MCP tools for a space.
func ListCommandsForSpace(db *store.DB, spaceID, repoRoot string) CommandsResponse {
	items := append([]CommandItem{}, BuiltinCommands()...)
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		root = "."
	}
	if list, err := skills.ScanRepo(root); err == nil && list != nil {
		for _, sk := range list.Items {
			name := strings.TrimSpace(sk.Name)
			if name == "" {
				name = strings.TrimSpace(sk.ID)
			}
			if name == "" {
				continue
			}
			if !strings.HasPrefix(name, "/") {
				name = "/" + name
			}
			desc := strings.TrimSpace(sk.Description)
			if desc == "" {
				desc = "skill"
			}
			items = append(items, CommandItem{
				Name: name, Description: desc, Source: CommandSourceSkill,
			})
		}
	}
	items = append(items, listMCPCommandItems(db, spaceID)...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Source != items[j].Source {
			return items[i].Source < items[j].Source
		}
		return items[i].Name < items[j].Name
	})
	return CommandsResponse{Items: items}
}

func listMCPCommandItems(db *store.DB, spaceID string) []CommandItem {
	if db == nil || strings.TrimSpace(spaceID) == "" {
		return nil
	}
	var rows []store.MCPTool
	q := db.Where("space_id = ?", spaceID).
		Where("status <> '' AND LOWER(status) <> ?", "disabled").
		Order("name asc")
	if err := q.Find(&rows).Error; err != nil {
		return nil
	}
	out := make([]CommandItem, 0, len(rows))
	for _, tool := range rows {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		if !strings.HasPrefix(name, "/") {
			name = "/" + name
		}
		server := strings.TrimSpace(tool.Server)
		desc := "MCP"
		if server != "" {
			desc = "MCP · " + server
		}
		out = append(out, CommandItem{
			Name: name, Description: desc, Source: CommandSourceMCP,
		})
	}
	return out
}

func findMCPToolByName(db *store.DB, spaceID, cmd string) (*store.MCPTool, error) {
	if db == nil || strings.TrimSpace(spaceID) == "" {
		return nil, gorm.ErrRecordNotFound
	}
	want := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(cmd), "/"))
	if want == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var rows []store.MCPTool
	if err := db.Where("space_id = ?", spaceID).
		Where("status <> '' AND LOWER(status) <> ?", "disabled").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		name := strings.ToLower(strings.TrimSpace(rows[i].Name))
		if name == want || strings.TrimPrefix(name, "mcp__") == want {
			return &rows[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func parseMCPArguments(args string) map[string]any {
	args = strings.TrimSpace(args)
	if args == "" {
		return map[string]any{}
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(args), &obj); err == nil && obj != nil {
		return obj
	}
	return map[string]any{}
}

func formatMCPResultText(res toolbus.Result) string {
	if !res.OK {
		msg := strings.TrimSpace(res.Error)
		if msg == "" {
			msg = "MCP tool call failed"
		}
		return "MCP 调用失败：" + msg
	}
	payload := any(res.Output)
	if res.Output != nil {
		if r, ok := res.Output["result"]; ok {
			payload = r
		}
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", payload)
	}
	text := string(b)
	const maxRunes = 4000
	if utf8.RuneCountInString(text) > maxRunes {
		runes := []rune(text)
		text = string(runes[:maxRunes]) + "…"
	}
	return text
}

func (s *Service) emitMCPToolEvents(view *View, tool *store.MCPTool, args map[string]any, res toolbus.Result) {
	if view == nil || tool == nil || strings.TrimSpace(view.RunID) == "" || s.events == nil {
		return
	}
	trace := firstNonEmpty(view.TraceID, view.RunID)
	_, _ = s.events.Append(view.RunID, trace, "tool.called", "info", map[string]any{
		"tool": "mcp.call", "name": tool.Name, "server": tool.Server, "arguments": args,
	}, events.WithVisibility(events.VisibilityUIOnly))
	payload := map[string]any{
		"tool": "mcp.call", "name": tool.Name, "ok": res.OK, "durationMs": res.DurationMs,
	}
	if res.OK {
		payload["output"] = res.Output
	} else {
		payload["error"] = res.Error
		if res.FailureClass != "" {
			payload["failureClass"] = res.FailureClass
		}
	}
	_, _ = s.events.Append(view.RunID, trace, "tool.result", "info", payload,
		events.WithVisibility(events.VisibilityUIOnly))
}

// ListModels returns builtin provider kinds used by applyProviderKind.
func ListModels() ModelsResponse {
	kinds := []string{"static", "acp_sdk", "execgo"}
	items := make([]ModelItem, 0, len(kinds))
	for _, k := range kinds {
		items = append(items, ModelItem{
			ID: k, Label: agentexec.DescribeKind(k), ProviderKind: k,
		})
	}
	return ModelsResponse{Items: items}
}

func normalizeCommandName(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	if !strings.HasPrefix(cmd, "/") {
		cmd = "/" + cmd
	}
	// first token only
	if i := strings.IndexAny(cmd, " \t"); i >= 0 {
		cmd = cmd[:i]
	}
	return strings.ToLower(cmd)
}

func normalizePermissionMode(mode string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "", PermissionReadOnly, PermissionWorkspaceWrite, PermissionFull:
		return m, nil
	default:
		return "", fmt.Errorf("invalid permissionMode %q (want read-only|workspace-write|full)", mode)
	}
}

func normalizeAgentMode(mode string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "", AgentModeCoding:
		return AgentModeCoding, nil
	case AgentModeGeneral:
		return AgentModeGeneral, nil
	default:
		return "", fmt.Errorf("invalid agentMode %q (want coding|general)", mode)
	}
}

func (s *Service) intentCommand(sessionID string, req IntentRequest) (*View, error) {
	view, err := s.Get(sessionID)
	if err != nil {
		return nil, err
	}
	if view.Status != StatusActive {
		return nil, fmt.Errorf("%w: session status %q", ErrIntentRejected, view.Status)
	}

	cmd := normalizeCommandName(req.Command)
	if cmd == "" {
		// allow command embedded in Prompt like "/help args"
		cmd = normalizeCommandName(req.Prompt)
	}
	if cmd == "" {
		return nil, fmt.Errorf("%w: command required", ErrIntentRejected)
	}
	args := strings.TrimSpace(req.Args)
	if args == "" && strings.TrimSpace(req.Prompt) != "" && normalizeCommandName(req.Prompt) != cmd {
		// leftover after command token in prompt
		rest := strings.TrimSpace(req.Prompt)
		if strings.HasPrefix(strings.ToLower(rest), cmd) {
			args = strings.TrimSpace(rest[len(cmd):])
		}
	}

	actor := firstNonEmpty(strings.TrimSpace(req.ActorID), view.CreatedBy, "session")
	if view.RunID != "" && s.events != nil {
		trace := firstNonEmpty(view.TraceID, view.RunID)
		_, _ = s.events.Append(view.RunID, trace, "session.command", "info", map[string]any{
			"sessionId": view.ID, "command": cmd, "args": args, "actorId": actor,
		}, events.WithVisibility(events.VisibilityUIOnly))
	}

	now := time.Now().UTC().Unix()
	turn := Turn{
		ID: "turn_" + uuid.NewString(), Prompt: cmd, CreatedAt: now,
	}
	if args != "" {
		turn.Prompt = cmd + " " + args
	}

	switch cmd {
	case "/help":
		view.Turns = append(view.Turns, turn)
		view.UpdatedAt = now
		catalog := ListCommandsForSpace(s.db, view.SpaceID, view.RepoRoot)
		s.emitAssistantReply(view, turn, formatHelpText(catalog), "command")
	case "/clear":
		view.Turns = []Turn{}
		view.Replies = []AssistantReply{}
		view.Turns = append(view.Turns, turn)
		view.UpdatedAt = now
		s.emitAssistantReply(view, turn, "已清空会话", "command")
	default:
		sk, lookupErr := lookupSkillCommand(view.RepoRoot, cmd)
		if lookupErr == nil && sk != nil {
			view.Turns = append(view.Turns, turn)
			view.UpdatedAt = now
			if !s.replyViaSkillLLM(view, turn, sk, args) {
				s.emitAssistantReply(view, turn, formatSkillLoadedReply(sk), "skill")
			}
			break
		}
		mcpTool, mcpErr := findMCPToolByName(s.db, view.SpaceID, cmd)
		if mcpErr != nil || mcpTool == nil {
			return nil, fmt.Errorf("%w: unknown command %q", ErrIntentRejected, cmd)
		}
		view.Turns = append(view.Turns, turn)
		view.UpdatedAt = now
		parsed := parseMCPArguments(args)
		res := toolbus.DefaultBus().Call(toolbus.Context{
			RunID:    view.RunID,
			TraceID:  view.TraceID,
			RepoRoot: view.RepoRoot,
		}, toolbus.CallRequest{
			Tool: "mcp.call",
			Args: map[string]any{
				"serverURL": mcpTool.Server,
				"name":      mcpTool.Name,
				"arguments": parsed,
			},
		})
		s.emitMCPToolEvents(view, mcpTool, parsed, res)
		s.emitAssistantReply(view, turn, formatMCPResultText(res), "mcp")
	}

	if err := s.save(view); err != nil {
		return nil, err
	}
	view.StreamURL = sessionStreamURL(view.ID)
	return view, nil
}

func formatHelpText(catalog CommandsResponse) string {
	var b strings.Builder
	b.WriteString("可用命令：\n")
	for _, it := range catalog.Items {
		b.WriteString("  ")
		b.WriteString(it.Name)
		if it.Description != "" {
			b.WriteString(" — ")
			b.WriteString(it.Description)
		}
		b.WriteString(" [")
		b.WriteString(it.Source)
		b.WriteString("]\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func lookupSkillCommand(repoRoot, cmd string) (*skills.Skill, error) {
	name := strings.TrimPrefix(strings.TrimSpace(cmd), "/")
	if name == "" {
		return nil, fmt.Errorf("empty skill name")
	}
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		root = "."
	}
	if sk, err := skills.Get(root, name); err == nil && sk != nil {
		return sk, nil
	}
	list, err := skills.ScanRepo(root)
	if err != nil || list == nil {
		return nil, fmt.Errorf("skill %q not found", name)
	}
	want := strings.ToLower(name)
	for i := range list.Items {
		sk := list.Items[i]
		id := strings.ToLower(strings.TrimSpace(sk.ID))
		nm := strings.ToLower(strings.TrimSpace(sk.Name))
		if id == want || nm == want {
			full, err := skills.Get(root, sk.ID)
			if err != nil {
				return &sk, nil
			}
			return full, nil
		}
	}
	return nil, fmt.Errorf("skill %q not found", name)
}

func formatSkillLoadedReply(sk *skills.Skill) string {
	if sk == nil {
		return "已加载技能，可用自然语言继续"
	}
	title := firstNonEmpty(strings.TrimSpace(sk.Name), strings.TrimSpace(sk.ID), "skill")
	body := strings.TrimSpace(sk.Body)
	const maxRunes = 600
	if body == "" {
		desc := strings.TrimSpace(sk.Description)
		if desc != "" {
			body = desc
		}
	}
	if n := len([]rune(body)); n > maxRunes {
		body = string([]rune(body)[:maxRunes]) + "…"
	}
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")
	if body != "" {
		b.WriteString(body)
		b.WriteString("\n\n")
	}
	b.WriteString("已加载技能，可用自然语言继续")
	return b.String()
}
