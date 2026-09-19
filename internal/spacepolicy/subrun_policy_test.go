package spacepolicy

import "testing"

func TestSubRunFromBodyJSON(t *testing.T) {
	p, err := SubRunFromBodyJSON(`{"subRun":{"maxDepth":1,"allowedTools":["read"],"tokenBudgetProxy":20}}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.MaxDepth != 1 || p.TokenBudgetProxy != 20 || len(p.AllowedTools) != 1 {
		t.Fatalf("%+v", p)
	}
	p, err = SubRunFromBodyJSON(`{}`)
	if err != nil || p.MaxDepth != 0 {
		t.Fatalf("empty %+v %v", p, err)
	}
	if _, err := SubRunFromBodyJSON(`{"subRun":{"maxDepth":-1}}`); err == nil {
		t.Fatal("expected negative maxDepth error")
	}
}
