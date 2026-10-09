package contextpack

import (
	"strings"
	"testing"
)

func TestBuildPrefixStableAndSkipsMemoryWithoutInject(t *testing.T) {
	in := Input{
		Issue:   "fix login",
		RAGRefs: []string{"file:b.go", "file:a.go"},
		Memories: []Hit{
			{ID: "m2", Title: "later"},
			{ID: "m1", Title: "earlier"},
		},
		Memory: []string{MemRetrieve},
	}
	a, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if a.Prefix != b.Prefix {
		t.Fatalf("prefix drifted\n%s\n%s", a.Prefix, b.Prefix)
	}
	if len(a.MemoryRefs) != 0 {
		t.Fatalf("memory refs=%v, want none without mem.inject", a.MemoryRefs)
	}
	for _, ref := range a.Refs {
		if ref == "memory:m1" || ref == "memory:m2" {
			t.Fatalf("pack refs include memory without inject: %v", a.Refs)
		}
	}
	if a.Refs[0] != "file:a.go" || a.Refs[1] != "file:b.go" {
		t.Fatalf("refs not sorted: %v", a.Refs)
	}
}

func TestBuildSkillPrefixOmitsBody(t *testing.T) {
	pack, err := Build(Input{
		Issue: "fix login",
		Skills: []SkillCard{{
			Name: "ash-demo", Description: "run when fixing login", Body: "SECRET_BODY_LINE",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pack.Prefix, "ash-demo") || strings.Contains(pack.Prefix, "SECRET_BODY_LINE") {
		t.Fatalf("prefix=%s", pack.Prefix)
	}
}

func TestBuildInjectsDeclaredMemoryOnly(t *testing.T) {
	pack, err := Build(Input{
		Issue: "fix login",
		Memories: []Hit{
			{ID: "m2", Title: "later"},
			{ID: "m1", Title: "earlier"},
		},
		Memory: []string{MemRetrieve, MemInject},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pack.MemoryRefs) != 2 || pack.MemoryRefs[0] != "memory:m1" || pack.MemoryRefs[1] != "memory:m2" {
		t.Fatalf("memory refs=%v", pack.MemoryRefs)
	}
	for _, ref := range pack.MemoryRefs {
		if !contains(pack.Refs, ref) {
			t.Fatalf("memory ref %s missing from pack refs %v", ref, pack.Refs)
		}
	}
}

func TestBuildRejectsInjectWithoutRetrieve(t *testing.T) {
	_, err := Build(Input{Memory: []string{MemInject}})
	if err == nil {
		t.Fatal("expected inject without retrieve to fail")
	}
}
