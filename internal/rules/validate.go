package rules

import (
	"fmt"
	"strings"
)

var (
	validPolicyProfiles = map[string]struct{}{
		"default": {}, "strict": {}, "hotfix": {}, "security": {},
	}
	validCheckpointStrategies = map[string]struct{}{
		"per_step": {}, "per_tool": {}, "manual": {},
	}
	validStepKinds = map[string]struct{}{
		"llm": {}, "agent": {}, "tool_chain": {}, "human": {}, "verify": {},
	}
	validHookPolicies = map[string]struct{}{
		"enforce": {}, "observe": {},
	}
	validHookEvents = map[string]struct{}{
		"tool.called": {}, "tool.result": {}, "run.before_finish": {},
		"run.started": {}, "step.started": {}, "step.finished": {},
		"policy.denied": {}, "memory.candidate_created": {},
	}
	validMissingCitationActions = map[string]struct{}{
		"block": {}, "human_confirm": {},
	}
)

// Validate performs semantic validation on a parsed Document.
func Validate(doc *Document) ValidationResult {
	res := ValidationResult{OK: true, Doc: doc}
	if doc == nil {
		return fail(res, "$", "NULL_DOCUMENT", "document is nil")
	}

	if doc.Version != Version {
		res = fail(res, "$.version", "INVALID_VERSION", fmt.Sprintf("version must be %q, got %q", Version, doc.Version))
	}

	sc := doc.Scenario
	if sc.Name == "" {
		res = fail(res, "$.scenario.name", "REQUIRED", "scenario.name is required")
	}
	if sc.ScenarioVersion == "" {
		res = fail(res, "$.scenario.scenarioVersion", "REQUIRED", "scenario.scenarioVersion is required")
	}
	profile := sc.PolicyProfile
	if profile == "" {
		profile = "default"
	} else if _, ok := validPolicyProfiles[profile]; !ok {
		res = fail(res, "$.scenario.policyProfile", "INVALID_ENUM", fmt.Sprintf("unknown policyProfile %q", profile))
	}
	if sc.Sandbox != nil && sc.Sandbox.MinMode != "" {
		switch sc.Sandbox.MinMode {
		case "off", "read-only", "workspace-write", "isolated":
		default:
			res = fail(res, "$.scenario.sandbox.minMode", "INVALID_ENUM", fmt.Sprintf("unknown sandbox.minMode %q", sc.Sandbox.MinMode))
		}
	}

	if sc.Checkpoint != nil {
		if _, ok := validCheckpointStrategies[sc.Checkpoint.Strategy]; !ok {
			res = fail(res, "$.scenario.checkpoint.strategy", "INVALID_ENUM", fmt.Sprintf("unknown checkpoint strategy %q", sc.Checkpoint.Strategy))
		}
		if sc.Checkpoint.Retain < 0 {
			res = fail(res, "$.scenario.checkpoint.retain", "INVALID_VALUE", "retain must be >= 0")
		}
	}

	if len(sc.Steps) == 0 {
		res = fail(res, "$.scenario.steps", "REQUIRED", "at least one step is required")
	}

	stepIDs := map[string]struct{}{}
	stepRoles := map[string]struct{}{}
	for i, st := range sc.Steps {
		p := fmt.Sprintf("$.scenario.steps[%d]", i)
		if st.ID == "" {
			res = fail(res, p+".id", "REQUIRED", "step id is required")
			continue
		}
		if _, dup := stepIDs[st.ID]; dup {
			res = fail(res, p+".id", "DUPLICATE", fmt.Sprintf("duplicate step id %q", st.ID))
		}
		stepIDs[st.ID] = struct{}{}
		if st.Role != "" {
			stepRoles[st.Role] = struct{}{}
		}
		if st.Role == "" {
			res = fail(res, p+".role", "REQUIRED", "step role is required")
		}
		if _, ok := validStepKinds[st.Kind]; !ok {
			res = fail(res, p+".kind", "INVALID_ENUM", fmt.Sprintf("step kind must be llm|agent|tool_chain|human|verify, got %q", st.Kind))
		}
		if st.Kind == "verify" {
			if st.Verify == nil || len(st.Verify.Checks) == 0 {
				res = fail(res, p+".verify.checks", "REQUIRED", "verify steps require verify.checks")
			} else if st.Verify.OnFail != "" && st.Verify.OnFail != "fail" && st.Verify.OnFail != "improve" {
				res = fail(res, p+".verify.onFail", "INVALID_ENUM", "verify.onFail must be fail|improve")
			}
		} else {
			switch st.Kind {
			case "llm":
				if st.PromptRef == "" {
					res = fail(res, p+".promptRef", "REQUIRED", "llm step requires promptRef")
				}
			case "agent":
				if st.Agent == nil || st.Agent.Adapter == "" {
					res = fail(res, p+".agent.adapter", "REQUIRED", "agent step requires agent.adapter")
				}
			case "tool_chain":
				if len(st.Chain) == 0 {
					res = fail(res, p+".chain", "REQUIRED", "tool_chain step requires chain")
				}
				for j, item := range st.Chain {
					if item.Tool == "" {
						res = fail(res, fmt.Sprintf("%s.chain[%d].tool", p, j), "REQUIRED", "tool name required")
					}
				}
			}
		}
		if st.RAG != nil && st.RAG.OnMissingCitations != "" {
			if _, ok := validMissingCitationActions[st.RAG.OnMissingCitations]; !ok {
				res = fail(res, p+".rag.onMissingCitations", "INVALID_ENUM", "onMissingCitations must be block|human_confirm")
			}
		}
		if sc.Roles != nil && st.Role != "" {
			if _, ok := sc.Roles[st.Role]; !ok {
				res = fail(res, p+".role", "UNKNOWN_ROLE", fmt.Sprintf("role %q not declared in scenario.roles", st.Role))
			}
		}
	}

	gateIDs := map[string]struct{}{}
	for i, g := range sc.Gates {
		p := fmt.Sprintf("$.scenario.gates[%d]", i)
		if g.ID == "" {
			res = fail(res, p+".id", "REQUIRED", "gate id is required")
		} else if _, dup := gateIDs[g.ID]; dup {
			res = fail(res, p+".id", "DUPLICATE", fmt.Sprintf("duplicate gate id %q", g.ID))
		} else {
			gateIDs[g.ID] = struct{}{}
		}
		if g.When == "" {
			res = fail(res, p+".when", "REQUIRED", "gate when is required")
		} else if err := validateGateWhen(g.When, stepIDs, stepRoles); err != nil {
			res = fail(res, p+".when", "INVALID_REFERENCE", err.Error())
		}
		hasTool := g.Check.Tool != ""
		hasArtifact := g.Check.Artifact != ""
		if hasTool == hasArtifact {
			res = fail(res, p+".check", "XOR_REQUIRED", "gate check must specify exactly one of tool or artifact")
		}
	}

	for i, h := range doc.Hooks {
		p := fmt.Sprintf("$.hooks[%d]", i)
		if h.ID == "" {
			res = fail(res, p+".id", "REQUIRED", "hook id is required")
		}
		if h.On == "" {
			res = fail(res, p+".on", "REQUIRED", "hook on is required")
		} else if _, ok := validHookEvents[h.On]; !ok {
			res = fail(res, p+".on", "INVALID_EVENT", fmt.Sprintf("hook event %q not in allowlist", h.On))
		}
		if h.Policy == "" {
			res = fail(res, p+".policy", "REQUIRED", "hook policy is required")
		} else if _, ok := validHookPolicies[h.Policy]; !ok {
			res = fail(res, p+".policy", "INVALID_ENUM", fmt.Sprintf("hook policy must be enforce|observe, got %q", h.Policy))
		}
		for j, r := range h.Rules {
			if len(r.Match) == 0 {
				res = fail(res, fmt.Sprintf("%s.rules[%d].match", p, j), "REQUIRED", "hook rule match is required")
			}
		}
	}

	if sc.Inputs != nil {
		for i, in := range sc.Inputs.Required {
			if in == "" {
				res = fail(res, fmt.Sprintf("$.scenario.inputs.required[%d]", i), "INVALID_VALUE", "empty input name")
			}
		}
	}

	return res
}

func validateGateWhen(when string, stepIDs, stepRoles map[string]struct{}) error {
	const prefix = "before.step."
	if strings.HasPrefix(when, prefix) {
		ref := strings.TrimPrefix(when, prefix)
		if ref == "" {
			return fmt.Errorf("empty step reference in %q", when)
		}
		if _, ok := stepIDs[ref]; !ok {
			return fmt.Errorf("gate references unknown step %q", ref)
		}
		return nil
	}
	const rolePrefix = "before.role."
	if strings.HasPrefix(when, rolePrefix) {
		ref := strings.TrimPrefix(when, rolePrefix)
		if _, ok := stepRoles[ref]; !ok {
			return fmt.Errorf("gate references unknown role %q", ref)
		}
		return nil
	}
	if when == "run.before_finish" {
		return nil
	}
	return fmt.Errorf("unsupported when expression %q (use before.step.<id>, before.role.<role>, or run.before_finish)", when)
}

func fail(res ValidationResult, path, code, msg string) ValidationResult {
	res.OK = false
	res.Issues = append(res.Issues, ValidationIssue{Path: path, Code: code, Message: msg})
	return res
}
