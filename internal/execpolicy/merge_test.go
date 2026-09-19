package execpolicy

import "testing"

func TestMerge_StricterWins(t *testing.T) {
	a := Policy{
		Version: SchemaVersion,
		Network: NetworkCaps{Egress: NetworkAllow},
		FS:      FSCaps{Mode: FSWorkspaceWrite},
		Process: ProcessCaps{Exec: ProcessAllow},
	}
	b := Policy{
		Version: SchemaVersion,
		Network: NetworkCaps{Egress: NetworkDeny},
		FS:      FSCaps{Mode: FSReadOnly},
		Process: ProcessCaps{Exec: ProcessAsk},
	}
	got := Merge(a, b)
	if got.Network.Egress != NetworkDeny {
		t.Fatalf("network=%q want deny", got.Network.Egress)
	}
	if got.FS.Mode != FSReadOnly {
		t.Fatalf("fs=%q want read-only", got.FS.Mode)
	}
	if got.Process.Exec != ProcessAsk {
		t.Fatalf("process=%q want ask", got.Process.Exec)
	}
	if got.Version != SchemaVersion {
		t.Fatalf("version=%q", got.Version)
	}
}

func TestMerge_UnsetDoesNotLower(t *testing.T) {
	space := Policy{
		Version: SchemaVersion,
		Network: NetworkCaps{Egress: NetworkDeny},
		FS:      FSCaps{Mode: FSNone},
	}
	harness := Policy{} // empty — must not loosen space floors
	got := Merge(space, harness)
	if got.Network.Egress != NetworkDeny || got.FS.Mode != FSNone {
		t.Fatalf("got=%+v want space floors preserved", got)
	}
	if got.Process.Exec != "" {
		t.Fatalf("process should stay unset, got %q", got.Process.Exec)
	}
}

func TestMerge_BothEmpty(t *testing.T) {
	got := Merge(Policy{}, Policy{})
	if !got.Empty() {
		t.Fatalf("want empty, got %+v", got)
	}
}

func TestMerge_Symmetric(t *testing.T) {
	a := Policy{Network: NetworkCaps{Egress: NetworkAsk}, FS: FSCaps{Mode: FSUnrestricted}}
	b := Policy{Network: NetworkCaps{Egress: NetworkAllow}, FS: FSCaps{Mode: FSNone}, Process: ProcessCaps{Exec: ProcessDeny}}
	ab := Merge(a, b)
	ba := Merge(b, a)
	if ab != ba {
		t.Fatalf("Merge not symmetric: ab=%+v ba=%+v", ab, ba)
	}
	if ab.Network.Egress != NetworkAsk || ab.FS.Mode != FSNone || ab.Process.Exec != ProcessDeny {
		t.Fatalf("merged=%+v", ab)
	}
}
