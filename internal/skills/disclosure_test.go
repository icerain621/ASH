package skills

import (
	"strings"
	"testing"
)

func TestDisclosureOmitsBody(t *testing.T) {
	lines := DisclosureLines([]Skill{{
		Name:        "ash-demo",
		Description: "run when fixing login",
		Body:        "SECRET_BODY_LINE do not put this in the prefix",
	}})
	if len(lines) != 1 || !strings.Contains(lines[0], "ash-demo") || !strings.Contains(lines[0], "fixing login") {
		t.Fatalf("lines=%v", lines)
	}
	if strings.Contains(lines[0], "SECRET_BODY_LINE") {
		t.Fatalf("body leaked: %s", lines[0])
	}
}
