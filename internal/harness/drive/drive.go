package drive

import "errors"

// ErrDangerUnapproved blocks a danger tool that has no approval.
var ErrDangerUnapproved = errors.New("danger tool requires approval")

// ErrRetryable marks a tool failure the drive may retry.
var ErrRetryable = errors.New("retryable tool failure")

// Drive places tools, retries retryable failures, and records a checkpoint.
type Drive struct {
	MaxRetries     int
	DangerApproved map[string]bool
	checkpoint     int
}

// Allow rejects danger tools that are not approved.
func (d *Drive) Allow(name, risk string) error {
	if risk != "danger" {
		return nil
	}
	if d == nil || !d.DangerApproved[name] {
		return ErrDangerUnapproved
	}
	return nil
}

// Execute runs fn when the effect gate allows it.
func (d *Drive) Execute(name, risk string, fn func() error) error {
	if err := d.Allow(name, risk); err != nil {
		return err
	}
	retries := 0
	if d != nil && d.MaxRetries > 0 {
		retries = d.MaxRetries
	}
	var err error
	for attempt := 0; attempt <= retries; attempt++ {
		err = fn()
		if err == nil {
			if d != nil {
				d.checkpoint++
			}
			return nil
		}
		if !errors.Is(err, ErrRetryable) {
			return err
		}
	}
	return err
}

// Checkpoint is the count of successful placements.
func (d *Drive) Checkpoint() int {
	if d == nil {
		return 0
	}
	return d.checkpoint
}
