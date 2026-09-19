package execpolicy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParsePolicyJSON parses an ash.execpolicy.v1 document.
// Empty body → empty Policy. Non-empty version must equal SchemaVersion.
func ParsePolicyJSON(body string) (Policy, error) {
	body = strings.TrimSpace(body)
	if body == "" || body == "null" {
		return Policy{}, nil
	}
	var p Policy
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		return Policy{}, fmt.Errorf("execpolicy: %w", err)
	}
	if err := normalizeAndValidate(&p); err != nil {
		return Policy{}, err
	}
	return p, nil
}

// FromSpaceBodyJSON reads the optional "execPolicy" key from SpacePolicy BodyJSON.
// Missing key / null / empty body → empty Policy (no floor; does not lower isolation).
func FromSpaceBodyJSON(bodyJSON string) (Policy, error) {
	bodyJSON = strings.TrimSpace(bodyJSON)
	if bodyJSON == "" || bodyJSON == "{}" {
		return Policy{}, nil
	}
	var body struct {
		ExecPolicy json.RawMessage `json:"execPolicy"`
	}
	if err := json.Unmarshal([]byte(bodyJSON), &body); err != nil {
		return Policy{}, fmt.Errorf("space policy bodyJson: %w", err)
	}
	if len(body.ExecPolicy) == 0 || string(body.ExecPolicy) == "null" {
		return Policy{}, nil
	}
	return ParsePolicyJSON(string(body.ExecPolicy))
}

// FromHarnessSpecJSON extracts ExecPolicy from a Harness Profile SpecJSON body.
// Preference order:
//  1. top-level "execPolicy" object (extension; not part of ash.harness/v1 schema)
//  2. synthesize from spec.sandbox.{defaultMode,network} when present
//
// Missing both → empty Policy.
func FromHarnessSpecJSON(specJSON string) (Policy, error) {
	specJSON = strings.TrimSpace(specJSON)
	if specJSON == "" || specJSON == "{}" {
		return Policy{}, nil
	}
	var top struct {
		ExecPolicy json.RawMessage `json:"execPolicy"`
		Spec       *struct {
			Sandbox *struct {
				DefaultMode string `json:"defaultMode"`
				Network     string `json:"network"`
			} `json:"sandbox"`
		} `json:"spec"`
	}
	if err := json.Unmarshal([]byte(specJSON), &top); err != nil {
		return Policy{}, fmt.Errorf("harness specJson: %w", err)
	}
	if len(top.ExecPolicy) > 0 && string(top.ExecPolicy) != "null" {
		return ParsePolicyJSON(string(top.ExecPolicy))
	}
	if top.Spec == nil || top.Spec.Sandbox == nil {
		return Policy{}, nil
	}
	return PolicyFromSandboxHints(top.Spec.Sandbox.DefaultMode, top.Spec.Sandbox.Network)
}

// PolicyFromSandboxHints maps harness sandbox.defaultMode / sandbox.network into a Policy.
// Unknown or empty hints leave the corresponding field unset.
func PolicyFromSandboxHints(defaultMode, network string) (Policy, error) {
	p := Policy{Version: SchemaVersion}
	net := strings.ToLower(strings.TrimSpace(network))
	switch net {
	case "":
		// unset
	case NetworkDeny, NetworkAllow, NetworkAsk:
		p.Network.Egress = net
	default:
		return Policy{}, fmt.Errorf("execpolicy: harness sandbox.network %q invalid", network)
	}
	mode := strings.ToLower(strings.TrimSpace(defaultMode))
	switch mode {
	case "":
		// unset
	case "off":
		p.FS.Mode = FSUnrestricted
		if p.Process.Exec == "" {
			p.Process.Exec = ProcessAllow
		}
	case "read-only":
		p.FS.Mode = FSReadOnly
		p.Process.Exec = ProcessAllow
	case "workspace-write":
		p.FS.Mode = FSWorkspaceWrite
		p.Process.Exec = ProcessAllow
	case "isolated":
		// Isolated implies no host FS write + subprocess jail; network stays from hint.
		p.FS.Mode = FSNone
		p.Process.Exec = ProcessAllow
	default:
		return Policy{}, fmt.Errorf("execpolicy: harness sandbox.defaultMode %q invalid", defaultMode)
	}
	if p.Network.Egress == "" && p.FS.Mode == "" && p.Process.Exec == "" {
		return Policy{}, nil
	}
	return p, nil
}

func normalizeAndValidate(p *Policy) error {
	p.Version = strings.TrimSpace(p.Version)
	if p.Version != "" && p.Version != SchemaVersion {
		return fmt.Errorf("execpolicy: version must be %q, got %q", SchemaVersion, p.Version)
	}

	egress := strings.ToLower(strings.TrimSpace(p.Network.Egress))
	switch egress {
	case "", NetworkDeny, NetworkAsk, NetworkAllow:
		p.Network.Egress = egress
	default:
		return fmt.Errorf("execpolicy: network.egress %q invalid (want deny|ask|allow)", p.Network.Egress)
	}

	fsMode := strings.ToLower(strings.TrimSpace(p.FS.Mode))
	switch fsMode {
	case "", FSNone, FSReadOnly, FSWorkspaceWrite, FSUnrestricted:
		p.FS.Mode = fsMode
	default:
		return fmt.Errorf("execpolicy: fs.mode %q invalid (want none|read-only|workspace-write|unrestricted)", p.FS.Mode)
	}

	exec := strings.ToLower(strings.TrimSpace(p.Process.Exec))
	switch exec {
	case "", ProcessDeny, ProcessAsk, ProcessAllow:
		p.Process.Exec = exec
	default:
		return fmt.Errorf("execpolicy: process.exec %q invalid (want deny|ask|allow)", p.Process.Exec)
	}
	return nil
}
