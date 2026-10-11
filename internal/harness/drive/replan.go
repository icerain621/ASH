package drive

// DefaultMaxReplanTurns caps verify.onFail=replan proposals per run+step (F06).
const DefaultMaxReplanTurns = 3

// ReplanBudget reports whether another replan proposal is allowed.
func ReplanBudget(proposed, maxTurns int) (remaining int, ok bool) {
	if maxTurns < 1 {
		maxTurns = DefaultMaxReplanTurns
	}
	if proposed < 0 {
		proposed = 0
	}
	remaining = maxTurns - proposed
	if remaining < 0 {
		remaining = 0
	}
	return remaining, remaining > 0
}
