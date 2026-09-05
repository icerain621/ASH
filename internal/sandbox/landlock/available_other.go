//go:build !linux

package landlock

// Available reports whether Landlock can be used on this platform.
// Non-Linux builds always return false.
func Available() bool {
	return false
}
