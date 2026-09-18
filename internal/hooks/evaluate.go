package hooks

import "strings"

func Evaluate(cfg Config, ev Event, ctx ToolContext) Decision {
	for i, rule := range cfg.Rules {
		if rule.Event != ev {
			continue
		}
		if !toolMatches(rule.Tool, ctx.Tool) {
			continue
		}
		if r := strings.TrimSpace(rule.Risk); r != "" {
			if !strings.EqualFold(r, strings.TrimSpace(ctx.Risk)) {
				continue
			}
		}
		action := normalizeAction(rule.Action)
		if action == "" {
			continue
		}
		return Decision{
			Action:    action,
			Reason:    strings.TrimSpace(rule.Reason),
			RuleIndex: i,
		}
	}
	return Decision{Action: ActionAllow, RuleIndex: -1}
}

func toolMatches(pattern, tool string) bool {
	pattern = strings.TrimSpace(pattern)
	tool = strings.TrimSpace(tool)
	if pattern == "" || pattern == "*" {
		return true
	}
	if tool == "" {
		return false
	}
	if pattern == tool {
		return true
	}
	if strings.HasSuffix(pattern, "*") && !strings.HasPrefix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return prefix != "" && strings.HasPrefix(tool, prefix)
	}
	if strings.HasPrefix(pattern, "*") && !strings.HasSuffix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return suffix != "" && strings.HasSuffix(tool, suffix)
	}
	return false
}

func normalizeAction(a DecisionAction) DecisionAction {
	switch strings.ToLower(strings.TrimSpace(string(a))) {
	case string(ActionAllow):
		return ActionAllow
	case string(ActionDeny):
		return ActionDeny
	case string(ActionAsk):
		return ActionAsk
	default:
		return ""
	}
}
