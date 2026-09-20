package ingress

import (
	"context"
	"slices"
	"testing"
)

func TestKnownAdapterIDs(t *testing.T) {
	got := KnownAdapterIDs()
	want := []string{"null", "webhook-github"}
	if !slices.Equal(got, want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
	got[0] = "mutated"
	if KnownAdapterIDs()[0] != "null" {
		t.Fatal("KnownAdapterIDs must return a fresh slice")
	}
}

func TestFromEnvDefaultNull(t *testing.T) {
	prev := getenv
	t.Cleanup(func() { getenv = prev })
	getenv = func(string) string { return "" }
	if FromEnv().Name() != "null" {
		t.Fatal(FromEnv().Name())
	}
}

func TestFromEnvWebhook(t *testing.T) {
	prev := getenv
	t.Cleanup(func() { getenv = prev })
	getenv = func(string) string { return "webhook-github" }
	if FromEnv().Name() != "webhook-github" {
		t.Fatal(FromEnv().Name())
	}
}

func TestFromEnvUnknownNull(t *testing.T) {
	prev := getenv
	t.Cleanup(func() { getenv = prev })
	getenv = func(string) string { return "telegram" }
	if FromEnv().Name() != "null" {
		t.Fatal("unknown must fail-closed to null")
	}
}

func TestWebhookGitHubAccept(t *testing.T) {
	ev, err := (WebhookGitHub{}).Accept(context.Background(), RawInbound{
		Headers: map[string]string{
			"X-GitHub-Delivery": "d1",
			"X-GitHub-Event":    "workflow_run",
		},
		Body: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if ev.DeliveryID != "d1" || ev.Kind != "workflow_run" {
		t.Fatalf("%+v", ev)
	}
}

func TestNullAcceptErrors(t *testing.T) {
	if _, err := (Null{}).Accept(context.Background(), RawInbound{}); err == nil {
		t.Fatal("expected error")
	}
}
