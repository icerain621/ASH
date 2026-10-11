package plugins

import (
	"errors"
	"strings"
	"testing"
)

func TestRegisterToolAndApprove(t *testing.T) {
	h := NewBuiltinHost()
	if err := h.Register(Component{ID: "tool.custom", Kind: KindTool, Status: StatusCandidate}); err != nil {
		t.Fatal(err)
	}
	if err := h.BindProduction([]string{"tool.custom"}); err == nil || !errors.Is(err, ErrNotApproved) {
		t.Fatalf("candidate must not bind: %v", err)
	}
	if _, err := h.GetProduction("tool.custom"); err == nil || !errors.Is(err, ErrNotApproved) {
		t.Fatalf("GetProduction candidate: %v", err)
	}
	if _, err := h.GetProduction("tool.missing.xyz"); err == nil || !errors.Is(err, ErrUnknown) {
		t.Fatalf("GetProduction unknown: %v", err)
	}
	if err := h.Approve("tool.custom"); err != nil {
		t.Fatal(err)
	}
	if err := h.BindProduction([]string{"tool.custom"}); err != nil {
		t.Fatal(err)
	}
}

func TestBindProductionInjectRequiresRetrieve(t *testing.T) {
	h := NewBuiltinHost()
	if err := h.BindProduction([]string{MemInject}); err == nil || !errors.Is(err, ErrInvalidBind) {
		t.Fatalf("err=%v", err)
	}
}

func TestCannotDemoteBuiltin(t *testing.T) {
	h := NewBuiltinHost()
	err := h.Register(Component{ID: MemRetrieve, Kind: KindMemory, Status: StatusCandidate})
	if err == nil || !strings.Contains(err.Error(), "already approved") {
		t.Fatalf("err=%v", err)
	}
}

func TestReplaceMemoryRetrieveAfterApprove(t *testing.T) {
	h := NewBuiltinHost()
	// Custom id can replace behavior slot once approved alongside builtins.
	if err := h.Register(Component{ID: "mem.retrieve.alt", Kind: KindMemory, Status: StatusCandidate}); err != nil {
		t.Fatal(err)
	}
	_ = h.Approve("mem.retrieve.alt")
	if err := h.BindProduction([]string{"mem.retrieve.alt"}); err != nil {
		t.Fatal(err)
	}
	if len(h.List()) < 8 {
		t.Fatalf("list=%d", len(h.List()))
	}
}

func TestPluginHostRejectsDirectApprovedRegister(t *testing.T) {
	h := NewBuiltinHost()
	if err := h.Register(Component{ID: "loop.custom", Kind: KindLoopPolicy, Status: StatusApproved}); err != nil {
		t.Fatal(err)
	}
	got, ok := h.Get("loop.custom")
	if !ok || got.Status != StatusCandidate {
		t.Fatalf("register must force candidate, got=%+v", got)
	}
	if err := h.BindProduction([]string{"loop.custom"}); err == nil {
		t.Fatal("candidate must not bind")
	}
	if err := h.Approve("loop.custom"); err != nil {
		t.Fatal(err)
	}
	if err := h.Register(Component{ID: "loop.custom", Kind: KindLoopPolicy}); err == nil {
		t.Fatal("must not demote approved via Register")
	}
	if err := h.BindProduction([]string{"loop.custom", "reviewer.default"}); err != nil {
		t.Fatal(err)
	}
}
