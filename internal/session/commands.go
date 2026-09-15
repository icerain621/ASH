package session

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/skills"
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
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Source != items[j].Source {
			return items[i].Source < items[j].Source
		}
		return items[i].Name < items[j].Name
	})
	return CommandsResponse{Items: items}
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
		s.emitAssistantReply(view, turn, formatHelpText(ListCommands(view.RepoRoot)), "command")
	case "/clear":
		view.Turns = []Turn{}
		view.Replies = []AssistantReply{}
		view.Turns = append(view.Turns, turn)
		view.UpdatedAt = now
		s.emitAssistantReply(view, turn, "已清空会话", "command")
	default:
		return nil, fmt.Errorf("%w: unknown command %q", ErrIntentRejected, cmd)
	}

	if err := s.save(view); err != nil {
		return nil, err
	}
	view.StreamURL = streamURL(view.RunID)
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
