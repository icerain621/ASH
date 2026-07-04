package opsenv

import (
	"strings"
	"testing"
)

func TestLiveGateHints(t *testing.T) {
	t.Setenv("ASH_MIGRATE_E2E", "1")
	t.Setenv("ASH_EXECGO_E2E", "1")
	t.Setenv("ASH_DATABASE_APP_URL", "postgres://ash_app@localhost/ash")
	hints := LiveGateHints()
	joined := strings.Join(hints, "\n")
	for _, want := range []string{"ASH_MIGRATE_E2E=1", "ASH_EXECGO_E2E=1", "ASH_DATABASE_APP_URL"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("hints=%v missing %q", hints, want)
		}
	}
}
