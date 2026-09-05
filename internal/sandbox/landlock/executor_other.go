//go:build !linux

package landlock

import (
	"context"
	"fmt"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

// Executor is a stub on non-Linux platforms.
type Executor struct{}

// Dispatch always fails: Landlock is Linux-only.
func (Executor) Dispatch(ctx context.Context, req sandbox.DispatchRequest) (*sandbox.DispatchResult, error) {
	return nil, fmt.Errorf("landlock unsupported on this OS")
}
