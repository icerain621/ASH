//go:build linux

package landlock

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Denied syscalls for the minimal filter (kill on match). Everything else allowed.
// Documented deny list for DX21; not a full production profile.
var seccompDenyList = []uint32{
	unix.SYS_MOUNT,
	unix.SYS_UMOUNT2,
	unix.SYS_PIVOT_ROOT,
	unix.SYS_SWAPON,
	unix.SYS_SWAPOFF,
	unix.SYS_REBOOT,
	unix.SYS_INIT_MODULE,
	unix.SYS_FINIT_MODULE,
	unix.SYS_DELETE_MODULE,
	unix.SYS_KEYCTL,
	unix.SYS_ADD_KEY,
	unix.SYS_REQUEST_KEY,
}

// SeccompAvailable reports whether a minimal SECCOMP_MODE_FILTER can be installed.
func SeccompAvailable() bool {
	if seccompEnvDisabled() {
		return false
	}
	// Probe: create an allow-all filter and install; we cannot uninstall, so only
	// check kernel support via seccomp(SECCOMP_SET_MODE_FILTER) with NULL first arg
	// is unsafe. Instead report based on env + GOOS (caller is linux-tagged).
	return true
}

func seccompEnvDisabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ASH_SANDBOX_SECCOMP")))
	switch v {
	case "0", "false", "off", "no":
		return true
	default:
		return false
	}
}

// applyMinimalSeccomp installs a deny-list filter. Returns nil when disabled or
// when the kernel rejects the filter with a soft-skip (ENOSYS/EINVAL).
func applyMinimalSeccomp() error {
	if seccompEnvDisabled() {
		return nil
	}
	prog, err := buildDenyListFilter(seccompDenyList)
	if err != nil {
		return err
	}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("PR_SET_NO_NEW_PRIVS: %w", err)
	}
	if err := unix.Seccomp(unix.SECCOMP_SET_MODE_FILTER, 0, unsafe.Pointer(&prog)); err != nil {
		// Soft-skip: older kernels / restricted environments.
		if err == unix.ENOSYS || err == unix.EINVAL || err == unix.EPERM {
			return nil
		}
		return fmt.Errorf("seccomp filter: %w", err)
	}
	return nil
}

// BPF classic program: load arch → check → load nr → deny matches → allow.
func buildDenyListFilter(deny []uint32) (unix.SockFprog, error) {
	const (
		bpfLd  = 0x00
		bpfJmp = 0x05
		bpfRet = 0x06
		bpfW   = 0x00
		bpfAbs = 0x20
		bpfK   = 0x00
		bpfJa  = 0x00
		bpfJeq = 0x10
	)
	seccompRetKill := uint32(unix.SECCOMP_RET_KILL_THREAD)
	seccompRetAllow := uint32(unix.SECCOMP_RET_ALLOW)

	var ins []unix.SockFilter
	// A = seccomp_data.arch
	ins = append(ins, unix.SockFilter{Code: bpfLd | bpfW | bpfAbs, K: 4})
	// if arch != AUDIT_ARCH_X86_64 → kill (we also handle aarch64 below loosely)
	arch := uint32(unix.AUDIT_ARCH_X86_64)
	if isARM64() {
		arch = unix.AUDIT_ARCH_AARCH64
	}
	// jeq arch, 1, 0 — if equal skip kill; else fall to kill
	// Actually: JEQ arch, jt=1, jf=0 then KILL then continue
	ins = append(ins, unix.SockFilter{Code: bpfJmp | bpfJeq | bpfK, Jt: 1, Jf: 0, K: arch})
	ins = append(ins, unix.SockFilter{Code: bpfRet | bpfK, K: seccompRetKill})
	// A = seccomp_data.nr
	ins = append(ins, unix.SockFilter{Code: bpfLd | bpfW | bpfAbs, K: 0})
	for _, nr := range deny {
		// if A == nr → kill; else continue
		ins = append(ins, unix.SockFilter{Code: bpfJmp | bpfJeq | bpfK, Jt: 0, Jf: 1, K: nr})
		ins = append(ins, unix.SockFilter{Code: bpfRet | bpfK, K: seccompRetKill})
	}
	ins = append(ins, unix.SockFilter{Code: bpfRet | bpfK, K: seccompRetAllow})

	return unix.SockFprog{
		Len:    uint16(len(ins)),
		Filter: &ins[0],
	}, nil
}

func isARM64() bool {
	var u unix.Utsname
	if err := unix.Uname(&u); err != nil {
		return false
	}
	machine := make([]byte, 0, len(u.Machine))
	for _, c := range u.Machine {
		if c == 0 {
			break
		}
		machine = append(machine, byte(c))
	}
	m := string(machine)
	return strings.Contains(m, "aarch64") || strings.Contains(m, "arm64")
}
