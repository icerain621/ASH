//go:build linux

package landlock

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Sentinel argv used when re-exec'ing the current binary as a Landlock child.
const childArgv0 = "__ASH_LANDLOCK__"

func init() {
	if len(os.Args) < 4 || os.Args[1] != childArgv0 {
		return
	}
	root := os.Args[2]
	program := os.Args[3]
	args := os.Args[4:]
	if err := becomeSandboxedChild(root, program, args); err != nil {
		fmt.Fprintf(os.Stderr, "ash landlock child: %v\n", err)
		os.Exit(127)
	}
}

func becomeSandboxedChild(root, program string, args []string) error {
	runtime.LockOSThread()
	if err := applyLandlockFS(root); err != nil {
		return err
	}
	if err := applyMinimalSeccomp(); err != nil {
		return fmt.Errorf("seccomp: %w", err)
	}
	env := os.Environ()
	argv := append([]string{program}, args...)
	return unix.Exec(program, argv, env)
}

func applyLandlockFS(root string) error {
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return err
	}
	abi, ok := probeLandlockABI()
	if !ok || abi < 1 {
		return fmt.Errorf("landlock unavailable")
	}
	handled := accessFSForABI(abi)
	attr := unix.LandlockRulesetAttr{Access_fs: handled}
	// Size 8 = Access_fs only for ABI compatibility with older kernels.
	fd, _, errno := unix.Syscall(
		unix.SYS_LANDLOCK_CREATE_RULESET,
		uintptr(unsafe.Pointer(&attr)),
		8,
		0,
	)
	if errno != 0 {
		return fmt.Errorf("landlock_create_ruleset: %w", errno)
	}
	ruleset := int(fd)
	defer func() { _ = unix.Close(ruleset) }()

	rw := handled
	ro := uint64(unix.LANDLOCK_ACCESS_FS_EXECUTE |
		unix.LANDLOCK_ACCESS_FS_READ_FILE |
		unix.LANDLOCK_ACCESS_FS_READ_DIR)

	if err := addPathBeneath(ruleset, absRoot, rw); err != nil {
		return fmt.Errorf("landlock allow repoRoot: %w", err)
	}
	for _, p := range []string{"/usr", "/bin", "/lib", "/lib64", "/etc", "/dev", "/proc", "/tmp", "/var/tmp"} {
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			continue
		}
		_ = addPathBeneath(ruleset, p, ro&handled)
	}

	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("PR_SET_NO_NEW_PRIVS: %w", err)
	}
	_, _, errno = unix.Syscall(unix.SYS_LANDLOCK_RESTRICT_SELF, uintptr(ruleset), 0, 0)
	if errno != 0 {
		return fmt.Errorf("landlock_restrict_self: %w", errno)
	}
	return nil
}

func accessFSForABI(abi int) uint64 {
	bits := uint64(
		unix.LANDLOCK_ACCESS_FS_EXECUTE |
			unix.LANDLOCK_ACCESS_FS_WRITE_FILE |
			unix.LANDLOCK_ACCESS_FS_READ_FILE |
			unix.LANDLOCK_ACCESS_FS_READ_DIR |
			unix.LANDLOCK_ACCESS_FS_REMOVE_DIR |
			unix.LANDLOCK_ACCESS_FS_REMOVE_FILE |
			unix.LANDLOCK_ACCESS_FS_MAKE_CHAR |
			unix.LANDLOCK_ACCESS_FS_MAKE_DIR |
			unix.LANDLOCK_ACCESS_FS_MAKE_REG |
			unix.LANDLOCK_ACCESS_FS_MAKE_SOCK |
			unix.LANDLOCK_ACCESS_FS_MAKE_FIFO |
			unix.LANDLOCK_ACCESS_FS_MAKE_BLOCK |
			unix.LANDLOCK_ACCESS_FS_MAKE_SYM,
	)
	if abi >= 2 {
		bits |= unix.LANDLOCK_ACCESS_FS_REFER
	}
	if abi >= 3 {
		bits |= unix.LANDLOCK_ACCESS_FS_TRUNCATE
	}
	if abi >= 4 {
		bits |= unix.LANDLOCK_ACCESS_FS_IOCTL_DEV
	}
	return bits
}

func addPathBeneath(rulesetFD int, path string, access uint64) error {
	if access == 0 {
		return nil
	}
	dirFD, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(dirFD) }()
	rule := unix.LandlockPathBeneathAttr{
		Allowed_access: access,
		Parent_fd:      int32(dirFD),
	}
	_, _, errno := unix.Syscall6(
		unix.SYS_LANDLOCK_ADD_RULE,
		uintptr(rulesetFD),
		uintptr(unix.LANDLOCK_RULE_PATH_BENEATH),
		uintptr(unsafe.Pointer(&rule)),
		0, 0, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
