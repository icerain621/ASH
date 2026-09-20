package notify

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

func TestFromEnvDefaultNull(t *testing.T) {
	t.Setenv("ASH_NOTIFIER", "")
	n := FromEnv()
	if n.Name() != "null" {
		t.Fatalf("name=%s", n.Name())
	}
	if err := n.Notify(context.Background(), Event{Kind: "x", Title: "t"}); err != nil {
		t.Fatal(err)
	}
}

func TestFromEnvLog(t *testing.T) {
	t.Setenv("ASH_NOTIFIER", "log")
	n := FromEnv()
	if n.Name() != "log" {
		t.Fatalf("name=%s", n.Name())
	}
}

func TestFromEnvUnknownNull(t *testing.T) {
	t.Setenv("ASH_NOTIFIER", "telegram")
	n := FromEnv()
	if n.Name() != "null" {
		t.Fatalf("unknown must fail-closed to null, got %s", n.Name())
	}
}

func TestLogNotifyWrites(t *testing.T) {
	var buf bytes.Buffer
	n := Log{Logger: log.New(&buf, "", 0)}
	if err := n.Notify(context.Background(), Event{
		Kind: "waker.duty.ok", Title: "stale_run ok", SpaceID: "local", Body: "done",
		Meta: map[string]string{"dutyId": "wd_1"},
	}); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, want := range []string{"waker.duty.ok", "stale_run ok", "local", "dutyId"} {
		if !strings.Contains(got, want) {
			t.Fatalf("log missing %q: %q", want, got)
		}
	}
}

func TestRecorder(t *testing.T) {
	r := NewRecorder()
	_ = r.Notify(context.Background(), Event{Kind: "a", Title: "t", Meta: map[string]string{"k": "v"}})
	evs := r.Events()
	if len(evs) != 1 || evs[0].Kind != "a" || evs[0].Meta["k"] != "v" {
		t.Fatalf("%+v", evs)
	}
	evs[0].Meta["k"] = "mutated"
	if r.Events()[0].Meta["k"] != "v" {
		t.Fatal("recorder should copy meta")
	}
}
