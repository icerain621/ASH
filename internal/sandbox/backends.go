package sandbox

// KnownBackendIDs is the static catalog of sandbox backends ASH understands (VX35).
// It projects existing router/remote executors; it does not invent new vendors.
func KnownBackendIDs() []string {
	// Return a fresh slice so callers cannot mutate the catalog.
	return []string{
		"local",
		"landlock",
		"docker",
		"remote-mock",
		"remote-e2b",
	}
}
