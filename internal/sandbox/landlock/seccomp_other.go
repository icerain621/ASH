//go:build !linux

package landlock

// SeccompAvailable is always false off Linux.
func SeccompAvailable() bool { return false }
