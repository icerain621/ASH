package drive

import "testing"

func TestReplanBudget(t *testing.T) {
	rem, ok := ReplanBudget(0, 3)
	if !ok || rem != 3 {
		t.Fatalf("rem=%d ok=%v", rem, ok)
	}
	rem, ok = ReplanBudget(3, 3)
	if ok || rem != 0 {
		t.Fatalf("exhausted rem=%d ok=%v", rem, ok)
	}
	rem, ok = ReplanBudget(1, 0) // default max
	if !ok || rem != DefaultMaxReplanTurns-1 {
		t.Fatalf("default rem=%d ok=%v", rem, ok)
	}
}
