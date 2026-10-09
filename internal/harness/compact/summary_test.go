package compact

import (
	"strings"
	"testing"
)

func TestSummarizeUsesSourceText(t *testing.T) {
	short, n := Summarize("remember AUDIT-42", 32)
	if n != 1 || !strings.Contains(short, "AUDIT-42") {
		t.Fatalf("short=%q n=%d", short, n)
	}
	long := "login handler rejects empty password. " + strings.Repeat("x", 500)
	summary, chunks := Summarize(long, 32)
	if chunks < 2 || !strings.Contains(summary, "login handler") {
		t.Fatalf("chunks=%d summary=%q", chunks, summary)
	}
	if strings.Contains(summary, "lossy compaction stub") {
		t.Fatalf("summary is still a stub: %q", summary)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	summary, chunks := Summarize("  ", 32)
	if summary != "" || chunks != 0 {
		t.Fatalf("summary=%q chunks=%d", summary, chunks)
	}
}
