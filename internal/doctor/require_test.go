package doctor

import (
	"testing"
)

func TestRequireCases_passAndSkip(t *testing.T) {
	rep := &Report{
		Results: []CaseResult{
			{ID: "M3-04", Status: "pass", Message: "ok"},
			{ID: "M3-06", Status: "pass", Message: "skipped: set ASH_POSTGRES_RLS=1", Evidence: []Evidence{{Kind: "skipped", Ref: "ASH_POSTGRES_RLS"}}},
		},
	}
	if err := RequireCases(rep, []string{"M3-04"}, true); err != nil {
		t.Fatalf("M3-04: %v", err)
	}
	if err := RequireCases(rep, []string{"M3-06"}, true); err == nil {
		t.Fatal("expected M3-06 skip rejection")
	}
	if err := RequireCases(rep, []string{"M3-06"}, false); err != nil {
		t.Fatalf("M3-06 allow skip: %v", err)
	}
}
