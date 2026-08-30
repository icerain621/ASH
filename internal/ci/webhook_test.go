package ci

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestVerifyGitHubSignature(t *testing.T) {
	body := []byte(`{"action":"completed"}`)
	secret := "whsec_test"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !VerifyGitHubSignature(secret, body, sig) {
		t.Fatal("expected valid signature")
	}
	if VerifyGitHubSignature(secret, body, "sha256=deadbeef") {
		t.Fatal("expected invalid signature")
	}
	if VerifyGitHubSignature("", body, sig) {
		t.Fatal("empty secret must fail")
	}
}

func TestIngestGitHubWebhookDiagnosesFailureAndIdempotent(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, func(spaceID, secretID string) (string, error) {
		return "whsec_test", nil
	})
	now := time.Now().UTC()
	conn := store.RepoConnection{
		ID: "repo_conn_wh", SpaceID: "local", Provider: "github",
		Owner: "acer", Repo: "ash", DefaultBranch: "main", SecretID: "sec_wh",
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&conn).Error; err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"action": "completed",
		"workflow_run": map[string]any{
			"id": 4242, "name": "CI", "status": "completed", "conclusion": "failure",
			"run_attempt": 1, "html_url": "https://example.test/run/4242",
			"head_branch": "main", "head_sha": "abc123",
			"created_at": now.Format(time.RFC3339), "updated_at": now.Format(time.RFC3339),
		},
		"repository": map[string]any{
			"name": "ash", "owner": map[string]any{"login": "acer"},
		},
	}
	body, _ := json.Marshal(payload)
	ctx := context.Background()
	first, err := svc.IngestGitHubWebhook(ctx, WebhookIngestRequest{
		ConnectionID: conn.ID, DeliveryID: "delivery-1", EventName: "workflow_run", Body: body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Duplicate || first.Ignored || first.Diagnosis == nil {
		t.Fatalf("first=%+v want diagnosis", first)
	}
	if first.Diagnosis.RootCause != "test_failure" {
		t.Fatalf("rootCause=%q", first.Diagnosis.RootCause)
	}
	if first.ShouldStartRun {
		t.Fatal("autoRun false must not set ShouldStartRun")
	}

	second, err := svc.IngestGitHubWebhook(ctx, WebhookIngestRequest{
		ConnectionID: conn.ID, DeliveryID: "delivery-1", EventName: "workflow_run", Body: body, AutoRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicate {
		t.Fatalf("second=%+v want duplicate", second)
	}
	if second.Diagnosis == nil || second.Diagnosis.ID != first.Diagnosis.ID {
		t.Fatalf("duplicate should replay diagnosis id %q got %+v", first.Diagnosis.ID, second.Diagnosis)
	}
}

func TestIngestGitHubWebhookIgnoresSuccess(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, func(spaceID, secretID string) (string, error) { return "x", nil })
	now := time.Now().UTC()
	conn := store.RepoConnection{
		ID: "repo_conn_ok", SpaceID: "local", Provider: "github",
		Owner: "acer", Repo: "ash", DefaultBranch: "main", SecretID: "sec",
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&conn).Error; err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{
		"action": "completed",
		"workflow_run": map[string]any{
			"id": 7, "name": "CI", "status": "completed", "conclusion": "success",
			"run_attempt": 1, "html_url": "https://example.test/run/7",
			"head_branch": "main", "head_sha": "def",
		},
	})
	out, err := svc.IngestGitHubWebhook(context.Background(), WebhookIngestRequest{
		ConnectionID: conn.ID, DeliveryID: "d-ok", EventName: "workflow_run", Body: body, AutoRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Ignored || out.Reason != "conclusion_not_failure" || out.ShouldStartRun {
		t.Fatalf("out=%+v", out)
	}
}
