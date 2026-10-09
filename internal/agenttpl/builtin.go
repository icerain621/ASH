package agenttpl

func base(id string, tools, memory []string, maxTurns int) Manifest {
	return Manifest{
		ID: id, Version: Version, Loop: "react", Tools: tools,
		Compaction: "threshold", Sandbox: "workspace-write",
		Hooks: []string{"PreToolUse", "PostToolUse"}, MaxTurns: maxTurns, Memory: memory,
	}
}

// React retrieves and injects memory, and may place read or bash.
func React() Manifest {
	return base("tpl.react", []string{"read", "bash"}, []string{"mem.retrieve", "mem.inject"}, 8)
}

// PlanSolve retrieves while planning and injects on the execute turn.
func PlanSolve() Manifest {
	m := base("tpl.plan-solve", []string{"read", "edit"}, []string{"mem.retrieve", "mem.inject"}, 8)
	m.PlanMemory = []string{"mem.retrieve"}
	return m
}

// Reviewer references no write-class memory component.
func Reviewer() Manifest {
	return base("tpl.reviewer", []string{"read"}, nil, 4)
}

// MemoryCurator may propose consolidate and forget, which still require review.
func MemoryCurator() Manifest {
	return base("tpl.memory-curator", []string{"read"}, []string{"mem.retrieve", "mem.consolidate", "mem.forget"}, 6)
}

// Research retrieves, injects, and records feedback.
func Research() Manifest {
	return base("tpl.research", []string{"read", "rag.query"}, []string{"mem.retrieve", "mem.inject", "mem.feedback"}, 8)
}
