package toolbus

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	runtimeHostPattern       = regexp.MustCompile(`^[A-Za-z0-9*_.-]+(:[0-9]{1,5})?$`)
	runtimeSecretNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,127}$`)
	runtimeSecretEnvPattern  = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,127}$`)
)

func registerRuntimeTools(r *Registry) {
	r.Register("runtime.command", RiskDanger, runtimeCommand)
}

func runtimeCommand(ctx Context, args map[string]any) (map[string]any, error) {
	program := firstStringArg(args, "program", "cmd")
	if strings.TrimSpace(program) == "" {
		return nil, runtimeError{class: "schema", msg: "program is required"}
	}
	timeoutMs := int64Arg(args, "timeoutMs", 30000)
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if timeoutMs > 300000 {
		return nil, runtimeError{class: "policy", msg: fmt.Sprintf("timeoutMs %d exceeds max 300000", timeoutMs)}
	}
	sandboxProfile, err := runtimeSandboxProfile(args)
	if err != nil {
		return nil, runtimeError{class: "policy", msg: err.Error()}
	}
	networkPolicy, allowedHosts, err := runtimeNetworkPolicy(args)
	if err != nil {
		return nil, runtimeError{class: "policy", msg: err.Error()}
	}
	if sandboxProfile == "process" && networkPolicy != "default" {
		return nil, runtimeError{class: "policy", msg: "networkPolicy requires an isolated sandboxProfile"}
	}
	secretRefs, err := runtimeSecretRefs(args)
	if err != nil {
		return nil, runtimeError{class: "policy", msg: err.Error()}
	}
	if len(secretRefs) > 0 && sandboxProfile == "process" {
		return nil, runtimeError{class: "policy", msg: "secretRefs require an isolated sandboxProfile"}
	}
	if len(secretRefs) > 0 && networkPolicy == "default" {
		return nil, runtimeError{class: "policy", msg: "secretRefs require explicit networkPolicy deny or allowlist"}
	}
	resources, err := runtimeResources(args, sandboxProfile)
	if err != nil {
		return nil, runtimeError{class: "policy", msg: err.Error()}
	}
	cli := firstNonEmpty(os.Getenv("EXECGO_EXECGOCLI"), "execgocli")
	if _, err := exec.LookPath(cli); err != nil {
		return nil, runtimeError{class: "bridge_unavailable", msg: fmt.Sprintf("execgocli not found: %v", err)}
	}
	actionID := runtimeActionID(ctx.RunID, ctx.TraceID, program, args)
	request := map[string]any{
		"adapter":    "toolbus",
		"agent_id":   "ash-toolbus",
		"session_id": firstNonEmpty(ctx.RunID, ctx.TraceID, actionID),
		"action_id":  actionID,
		"action": map[string]any{
			"kind": "runtime.command",
			"input": map[string]any{
				"program": program,
				"args":    stringSliceArg(args, "args"),
				"cwd":     firstNonEmpty(firstStringArg(args, "cwd"), ctx.RepoRoot),
				"limits": map[string]any{
					"wall_time_ms": timeoutMs,
				},
				"resources": resources,
				"sandbox": map[string]any{
					"profile":     sandboxProfile,
					"network":     map[string]any{"policy": networkPolicy, "allowed_hosts": allowedHosts},
					"secret_refs": secretRefs,
				},
			},
		},
		"metadata": map[string]any{
			"source":  "ash.toolbus",
			"runId":   ctx.RunID,
			"traceId": ctx.TraceID,
			"tool":    "runtime.command",
		},
	}
	path, err := writeRuntimeAction(ctx.RunDir, actionID, request)
	if err != nil {
		return nil, runtimeError{class: "io", msg: err.Error()}
	}
	if _, _, err := execGoJSON(cli, "health"); err != nil {
		return nil, runtimeError{class: "bridge_unavailable", msg: fmt.Sprintf("execgo health: %v", err)}
	}
	if _, _, err := execGoJSON(cli, "tools"); err != nil {
		return nil, runtimeError{class: "bridge_unavailable", msg: fmt.Sprintf("execgo tools: %v", err)}
	}
	act, stdout, err := execGoJSON(cli, "act", "-file", path)
	if err != nil {
		return nil, runtimeError{class: "submit", msg: err.Error()}
	}
	taskID := firstString(act, "task_id")
	if taskID == "" {
		taskID = actionID
	}
	wait, waitOut, err := execGoJSON(cli, "wait", "-task-ids", taskID)
	stdout = joinText(stdout, waitOut)
	if err != nil {
		return nil, runtimeError{class: "wait", msg: err.Error()}
	}
	status := terminalStatus(wait)
	if status == "" {
		status = "success"
	}
	if status != "success" {
		return map[string]any{"taskId": taskID, "status": status, "stdout": trimText(stdout)}, runtimeError{
			class: "runtime_failed", msg: fmt.Sprintf("runtime task %s ended with status %s", taskID, status),
		}
	}
	return map[string]any{
		"taskId": taskID, "status": status, "actionId": actionID,
		"actionFile": path, "stdout": trimText(stdout),
		"policy": map[string]any{
			"sandboxProfile": sandboxProfile,
			"networkPolicy":  networkPolicy,
			"allowedHosts":   allowedHosts,
			"resources":      resources,
			"secretRefCount": len(secretRefs),
		},
	}, nil
}

type runtimeError struct {
	class string
	msg   string
}

func (e runtimeError) Error() string {
	return e.msg
}

func (e runtimeError) FailureClass() string {
	return "runtime_" + e.class
}

func writeRuntimeAction(runDir, actionID string, request map[string]any) (string, error) {
	if strings.TrimSpace(runDir) == "" {
		runDir = os.TempDir()
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(runDir, "tool-runtime-"+actionID+".json")
	b, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func execGoJSON(cli string, args ...string) (map[string]any, string, error) {
	cmd := exec.CommandContext(context.Background(), cli, args...)
	cmd.Env = append(os.Environ(),
		"EXECGO_URL="+firstNonEmpty(os.Getenv("EXECGO_URL"), "http://127.0.0.1:8080"),
		"EXECGO_RUNTIME_URL="+firstNonEmpty(os.Getenv("EXECGO_RUNTIME_URL"), "http://127.0.0.1:18080"),
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, stdout.String(), fmt.Errorf("%s %s: %w: %s", cli, strings.Join(args, " "), err, trimText(stderr.String()))
	}
	var envelope struct {
		OK    bool           `json:"ok"`
		Data  map[string]any `json:"data"`
		Error *struct {
			Message    string `json:"message"`
			StatusCode int    `json:"status_code"`
			Body       string `json:"body"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		return nil, stdout.String(), fmt.Errorf("parse execgocli JSON: %w: %s", err, trimText(stdout.String()))
	}
	if !envelope.OK {
		if envelope.Error != nil {
			return nil, stdout.String(), fmt.Errorf("%s (status=%d body=%s)", envelope.Error.Message, envelope.Error.StatusCode, envelope.Error.Body)
		}
		return nil, stdout.String(), fmt.Errorf("execgocli returned ok=false")
	}
	if envelope.Data == nil {
		envelope.Data = map[string]any{}
	}
	return envelope.Data, stdout.String(), nil
}

func stringSliceArg(args map[string]any, key string) []string {
	switch items := args[key].(type) {
	case []string:
		return append([]string{}, items...)
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

func runtimeSandboxProfile(args map[string]any) (string, error) {
	profile := strings.ToLower(strings.TrimSpace(firstNonEmpty(firstStringArg(args, "sandboxProfile"), "process")))
	switch profile {
	case "process", "container", "firecracker":
		return profile, nil
	default:
		return "", fmt.Errorf("sandboxProfile must be process|container|firecracker")
	}
}

func runtimeNetworkPolicy(args map[string]any) (string, []string, error) {
	policy := strings.ToLower(strings.TrimSpace(firstNonEmpty(firstStringArg(args, "networkPolicy"), "default")))
	switch policy {
	case "default", "deny", "allowlist":
	default:
		return "", nil, fmt.Errorf("networkPolicy must be default|deny|allowlist")
	}
	hosts := stringSliceArg(args, "allowedHosts")
	if policy != "allowlist" && len(hosts) > 0 {
		return "", nil, fmt.Errorf("allowedHosts requires networkPolicy allowlist")
	}
	if policy == "allowlist" && len(hosts) == 0 {
		return "", nil, fmt.Errorf("allowedHosts is required when networkPolicy is allowlist")
	}
	out := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = strings.ToLower(strings.TrimSpace(host))
		if host == "" || host == "*" || strings.Contains(host, "://") || strings.Contains(host, "/") || !runtimeHostPattern.MatchString(host) {
			return "", nil, fmt.Errorf("invalid allowed host %q", host)
		}
		if strings.Contains(host, "*") && !strings.HasPrefix(host, "*.") {
			return "", nil, fmt.Errorf("invalid allowed host wildcard %q", host)
		}
		if err := validateRuntimeHostPort(host); err != nil {
			return "", nil, err
		}
		out = append(out, host)
	}
	return policy, out, nil
}

func runtimeResources(args map[string]any, sandboxProfile string) (map[string]any, error) {
	memoryMB := int64Arg(args, "memoryMB", 0)
	if memoryMB < 0 || memoryMB > 32768 {
		return nil, fmt.Errorf("memoryMB must be between 0 and 32768")
	}
	cpuMillis := int64Arg(args, "cpuMillis", 0)
	if cpuMillis < 0 || cpuMillis > 32000 {
		return nil, fmt.Errorf("cpuMillis must be between 0 and 32000")
	}
	if sandboxProfile != "process" && (memoryMB == 0 || cpuMillis == 0) {
		return nil, fmt.Errorf("isolated sandboxProfile requires memoryMB and cpuMillis limits")
	}
	return map[string]any{"memory_mb": memoryMB, "cpu_millis": cpuMillis}, nil
}

func validateRuntimeHostPort(host string) error {
	idx := strings.LastIndex(host, ":")
	if idx < 0 {
		return nil
	}
	port, err := strconv.Atoi(host[idx+1:])
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid allowed host port %q", host)
	}
	return nil
}

func runtimeSecretRefs(args map[string]any) ([]map[string]any, error) {
	raw, ok := args["secretRefs"]
	if !ok || raw == nil {
		return nil, nil
	}
	var items []any
	switch v := raw.(type) {
	case []any:
		items = v
	case []string:
		items = make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
	default:
		return nil, fmt.Errorf("secretRefs must be an array")
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		ref, err := runtimeSecretRef(item)
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

func runtimeSecretRef(item any) (map[string]any, error) {
	switch v := item.(type) {
	case string:
		name := strings.TrimSpace(v)
		if !runtimeSecretNamePattern.MatchString(name) {
			return nil, fmt.Errorf("invalid secret ref name %q", name)
		}
		return map[string]any{"name": name, "env": secretEnvName(name)}, nil
	case map[string]any:
		for key := range v {
			if key != "name" && key != "env" {
				return nil, fmt.Errorf("secretRefs only allow name and env fields")
			}
		}
		name := firstString(v, "name")
		env := firstNonEmpty(firstString(v, "env"), secretEnvName(name))
		if !runtimeSecretNamePattern.MatchString(name) {
			return nil, fmt.Errorf("invalid secret ref name %q", name)
		}
		if !runtimeSecretEnvPattern.MatchString(env) {
			return nil, fmt.Errorf("invalid secret env %q", env)
		}
		return map[string]any{"name": name, "env": env}, nil
	default:
		return nil, fmt.Errorf("secretRefs entries must be strings or objects")
	}
}

func secretEnvName(name string) string {
	name = strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(name, ".", "_"), "-", "_"))
	if !runtimeSecretEnvPattern.MatchString(name) {
		return "ASH_SECRET"
	}
	return name
}

func runtimeActionID(runID, traceID, program string, args map[string]any) string {
	sum := sha256.Sum256([]byte(runID + ":" + traceID + ":" + program + ":" + fmt.Sprintf("%v", args)))
	return "ash-runtime-" + hex.EncodeToString(sum[:])[:20]
}

func terminalStatus(data map[string]any) string {
	if tasks, ok := data["tasks"].([]any); ok && len(tasks) > 0 {
		if task, ok := tasks[0].(map[string]any); ok {
			if st, ok := task["status"].(string); ok {
				return st
			}
			if st, ok := task["state"].(string); ok {
				return st
			}
		}
	}
	if st, ok := data["status"].(string); ok {
		return st
	}
	return ""
}

func firstString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func joinText(left, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	return left + "\n" + right
}

func trimText(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 4000 {
		return s
	}
	return s[:4000] + "\n...<truncated>"
}
