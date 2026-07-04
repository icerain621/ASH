package opsenv

import "os"

type liveGate struct {
	env  string
	hint string
}

var liveGateDefs = []liveGate{
	{env: "ASH_CI_FIXTURE", hint: "CI sync uses fixture provider (no GitHub API)"},
	{env: "ASH_MIGRATE_E2E", hint: "M3-04 live migrate verify"},
	{env: "ASH_EXECGO_E2E", hint: "M3-05 ExecGo/Codex live smoke"},
	{env: "ASH_POSTGRES_RLS", hint: "M3-06 Postgres RLS policies"},
}

// LiveGateHints lists enabled ASH_* live gates (names only, no secret values).
func LiveGateHints() []string {
	out := make([]string, 0, len(liveGateDefs)+2)
	for _, g := range liveGateDefs {
		if os.Getenv(g.env) == "1" {
			out = append(out, g.env+"=1 · "+g.hint)
		}
	}
	if os.Getenv("ASH_DATABASE_APP_URL") != "" {
		out = append(out, "ASH_DATABASE_APP_URL set · M3-07 ash_app role")
	}
	if os.Getenv("ASH_SCHEMA_MODE") == "sql" {
		out = append(out, "ASH_SCHEMA_MODE=sql · M3-08 golang-migrate")
	}
	return out
}
