package security

import (
	"strings"
	"testing"
)

func TestFindLeaksDetectsBearerToken(t *testing.T) {
	findings := FindLeaks("audit", "aud_1", `{"authorization":"Bearer sk-live-abcdefghijklmnopqrst"}`)
	if len(findings) == 0 {
		t.Fatal("expected leak findings")
	}
}

func TestRedactJSONMasksSensitiveKeys(t *testing.T) {
	raw := `{"password":"super-secret-value","note":"ok"}`
	redacted := RedactJSON(raw)
	if redacted == raw || !strings.Contains(redacted, "***REDACTED***") {
		t.Fatalf("redacted=%q", redacted)
	}
	if strings.Contains(redacted, "super-secret-value") {
		t.Fatalf("secret value still present: %q", redacted)
	}
}
