package lenses

// Lens is one product observation view. All three read the same event spine.
type Lens string

const (
	LensGlobal Lens = "global"
	LensAgent  Lens = "agent"
	LensMemory Lens = "memory"
)

// Signal maps a component event onto one product lens.
type Signal struct {
	Component string
	EventType string
	Lens      Lens
}

// Catalog is the v7 P0 signal register. Unlisted events stay out of product lenses.
func Catalog() []Signal {
	return []Signal{
		{Component: "contextpack", EventType: "context.packed", Lens: LensAgent},
		{Component: "runtime", EventType: "agent.called", Lens: LensAgent},
		{Component: "mem.inject", EventType: "memory.injected", Lens: LensMemory},
		{Component: "mem.inject", EventType: "memory.hit_used", Lens: LensMemory},
		{Component: "mem.feedback", EventType: "memory.feedback", Lens: LensMemory},
		{Component: "compaction", EventType: "harness.compaction", Lens: LensAgent},
		{Component: "session", EventType: "session.steer", Lens: LensAgent},
		{Component: "session", EventType: "session.follow_up", Lens: LensAgent},
		{Component: "runtime", EventType: "scenario.step_deprecated", Lens: LensAgent},
		{Component: "runtime", EventType: "template.finished", Lens: LensAgent},
		{Component: "mem.retrieve", EventType: "memory.query_failed", Lens: LensMemory},
		{Component: "mem.consolidate", EventType: "memory.consolidate_proposed", Lens: LensMemory},
		{Component: "mem.forget", EventType: "memory.forget_proposed", Lens: LensMemory},
	}
}

// Admit reports whether eventType is registered for lens.
func Admit(eventType string, lens Lens) bool {
	for _, sig := range Catalog() {
		if sig.EventType == eventType && sig.Lens == lens {
			return true
		}
	}
	return false
}
