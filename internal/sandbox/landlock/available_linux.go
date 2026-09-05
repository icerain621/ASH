//go:build linux

package landlock

import (
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// Available reports whether the running kernel supports Landlock.
// Probe is best-effort and never panics: uses landlock_create_ruleset
// ABI version query, with uname release ≥ 5.13 as a secondary hint when
// the syscall returns an unexpected zero ABI without errno.
func Available() bool {
	defer func() { _ = recover() }()
	if runtime.GOOS != "linux" {
		return false
	}
	if abi, ok := probeLandlockABI(); ok && abi >= 1 {
		return true
	}
	// Syscall missing/unsupported → false even on new kernels without Landlock compiled in.
	if _, ok := probeLandlockABI(); !ok {
		return false
	}
	return kernelReleaseAtLeast(5, 13)
}

func probeLandlockABI() (int, bool) {
	r0, _, errno := unix.Syscall(
		unix.SYS_LANDLOCK_CREATE_RULESET,
		0,
		0,
		uintptr(unix.LANDLOCK_CREATE_RULESET_VERSION),
	)
	if errno != 0 {
		return 0, false
	}
	return int(r0), true
}

func kernelReleaseAtLeast(major, minor int) bool {
	var uts unix.Utsname
	if err := unix.Uname(&uts); err != nil {
		return false
	}
	rel := utsReleaseString(uts.Release[:])
	parts := strings.SplitN(rel, ".", 3)
	if len(parts) < 2 {
		return false
	}
	maj, err1 := strconv.Atoi(digitsPrefix(parts[0]))
	min, err2 := strconv.Atoi(digitsPrefix(parts[1]))
	if err1 != nil || err2 != nil {
		return false
	}
	if maj != major {
		return maj > major
	}
	return min >= minor
}

func utsReleaseString(b []byte) string {
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	return string(b[:n])
}

func digitsPrefix(s string) string {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return s
	}
	return s[:i]
}
