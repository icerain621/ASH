// Package execpolicy defines ash.execpolicy.v1 capability declarations
// (network / fs / process) and stricter-wins merge (V6-D3).
// Sandbox ResolveSandboxMode wiring is EW15 — not here.
package execpolicy

import "strings"

// SchemaVersion is the only accepted non-empty Policy.version.
const SchemaVersion = "ash.execpolicy.v1"

// Network egress modes (stricter → looser): deny > ask > allow.
const (
	NetworkDeny  = "deny"
	NetworkAsk   = "ask"
	NetworkAllow = "allow"
)

// Filesystem modes (stricter → looser): none > read-only > workspace-write > unrestricted.
const (
	FSNone           = "none"
	FSReadOnly       = "read-only"
	FSWorkspaceWrite = "workspace-write"
	FSUnrestricted   = "unrestricted"
)

// Process exec modes (stricter → looser): deny > ask > allow.
const (
	ProcessDeny  = "deny"
	ProcessAsk   = "ask"
	ProcessAllow = "allow"
)

// NetworkCaps declares outbound network capability.
type NetworkCaps struct {
	// Egress: deny | ask | allow. Empty = unset (no floor from this policy).
	Egress string `json:"egress,omitempty"`
}

// FSCaps declares filesystem capability.
type FSCaps struct {
	// Mode: none | read-only | workspace-write | unrestricted. Empty = unset.
	Mode string `json:"mode,omitempty"`
}

// ProcessCaps declares process/exec capability.
type ProcessCaps struct {
	// Exec: deny | ask | allow. Empty = unset.
	Exec string `json:"exec,omitempty"`
}

// Policy is the ash.execpolicy.v1 document.
type Policy struct {
	Version string      `json:"version,omitempty"`
	Network NetworkCaps `json:"network,omitempty"`
	FS      FSCaps      `json:"fs,omitempty"`
	Process ProcessCaps `json:"process,omitempty"`
}

// Empty reports whether no capability floor is declared (and version is empty).
func (p Policy) Empty() bool {
	return strings.TrimSpace(p.Version) == "" &&
		strings.TrimSpace(p.Network.Egress) == "" &&
		strings.TrimSpace(p.FS.Mode) == "" &&
		strings.TrimSpace(p.Process.Exec) == ""
}
