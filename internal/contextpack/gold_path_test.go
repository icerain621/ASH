package contextpack

import (
	"strings"
	"testing"
)

func TestGoldPathPackHasMemoryAndRAG(t *testing.T) {
	pack, err := Build(Input{
		Issue:   "thin feature delivery",
		RAGRefs: []string{"file:scenarios/feature_delivery.yaml", "rag:chunk:login"},
		Memories: []Hit{
			{ID: "mem_login_fix", Title: "prior login fix"},
		},
		Memory: []string{MemRetrieve, MemInject},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pack.Prefix, "# evidence") || !strings.Contains(pack.Prefix, "# memory") {
		t.Fatalf("prefix missing sections: %s", pack.Prefix)
	}
	hasRAG, hasMem := false, false
	for _, ref := range pack.Refs {
		if strings.HasPrefix(ref, "file:") || strings.HasPrefix(ref, "rag:") {
			hasRAG = true
		}
		if strings.HasPrefix(ref, "memory:") {
			hasMem = true
		}
	}
	if !hasRAG || !hasMem {
		t.Fatalf("refs=%v want both RAG and memory", pack.Refs)
	}
	if len(pack.MemoryRefs) != 1 || pack.MemoryRefs[0] != "memory:mem_login_fix" {
		t.Fatalf("memoryRefs=%v", pack.MemoryRefs)
	}
}
