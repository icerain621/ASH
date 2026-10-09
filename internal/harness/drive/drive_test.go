package drive

import (
	"errors"
	"testing"
)

func TestDangerToolBlockedWithoutApproval(t *testing.T) {
	d := &Drive{}
	called := false
	err := d.Execute("bash", "danger", func() error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrDangerUnapproved) || called {
		t.Fatalf("err=%v called=%v", err, called)
	}
	if d.Checkpoint() != 0 {
		t.Fatalf("checkpoint=%d", d.Checkpoint())
	}
}

func TestApprovedDangerRunsAndCheckpoints(t *testing.T) {
	d := &Drive{DangerApproved: map[string]bool{"bash": true}}
	if err := d.Execute("bash", "danger", func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if d.Checkpoint() != 1 {
		t.Fatalf("checkpoint=%d", d.Checkpoint())
	}
}

func TestRetryThenCheckpoint(t *testing.T) {
	d := &Drive{MaxRetries: 1}
	tries := 0
	err := d.Execute("read", "safe", func() error {
		tries++
		if tries == 1 {
			return ErrRetryable
		}
		return nil
	})
	if err != nil || tries != 2 || d.Checkpoint() != 1 {
		t.Fatalf("err=%v tries=%d checkpoint=%d", err, tries, d.Checkpoint())
	}
}
