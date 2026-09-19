package execpolicy

import "strings"

// Merge returns the stricter floors of a and b on each capability axis (V6-D3).
// Unset fields do not contribute; the other side's value is kept.
// Version is set to SchemaVersion when the result is non-empty.
func Merge(a, b Policy) Policy {
	out := Policy{
		Network: NetworkCaps{Egress: stricterString(a.Network.Egress, b.Network.Egress, networkRank)},
		FS:      FSCaps{Mode: stricterString(a.FS.Mode, b.FS.Mode, fsRank)},
		Process: ProcessCaps{Exec: stricterString(a.Process.Exec, b.Process.Exec, processRank)},
	}
	if !out.Empty() {
		out.Version = SchemaVersion
	}
	return out
}

func stricterString(a, b string, rank func(string) int) string {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	ra, rb := rank(a), rank(b)
	// rank 0 = unset / unknown; higher = stricter
	if ra == 0 {
		return b
	}
	if rb == 0 {
		return a
	}
	if ra >= rb {
		return a
	}
	return b
}

func networkRank(v string) int {
	switch v {
	case NetworkDeny:
		return 3
	case NetworkAsk:
		return 2
	case NetworkAllow:
		return 1
	default:
		return 0
	}
}

func fsRank(v string) int {
	switch v {
	case FSNone:
		return 4
	case FSReadOnly:
		return 3
	case FSWorkspaceWrite:
		return 2
	case FSUnrestricted:
		return 1
	default:
		return 0
	}
}

func processRank(v string) int {
	switch v {
	case ProcessDeny:
		return 3
	case ProcessAsk:
		return 2
	case ProcessAllow:
		return 1
	default:
		return 0
	}
}
