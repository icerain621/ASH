package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestComplianceSecretScan(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	space := "space_secret_scan"
	_ = db.Create(&store.AuditPolicy{SpaceID: space, RetentionDays: 30, CreatedAt: now, UpdatedAt: now}).Error
	_ = db.Create(&store.AuditLog{
		ID: "aud_leak_1", SpaceID: space, EventType: "test.probe",
		PayloadJSON: `{"password":"plaintext-leak-value"}`, CreatedAt: now,
	}).Error

	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/secret-scan?limit=50", nil)
	req.Header.Set("X-ASH-Space-ID", space)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp SecretScanResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.LeakCount == 0 {
		t.Fatalf("leakCount=0 want >0, resp=%+v", resp)
	}
}

func TestComplianceExportIncludesSecretScan(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	space := "space_export_scan"
	_ = db.Create(&store.AuditPolicy{SpaceID: space, RetentionDays: 30, CreatedAt: now, UpdatedAt: now}).Error
	_ = db.Create(&store.AuditLog{
		ID: "aud_export_1", SpaceID: space, EventType: "test.export",
		PayloadJSON: `{"token":"export-leak-probe"}`, CreatedAt: now,
	}).Error

	body := []byte(`{"suite":"TR2"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ASH-Space-ID", space)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var row store.AuditExport
	if err := db.Order("created_at desc").First(&row, "space_id = ?", space).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != "completed" || row.StoreKey == "" {
		t.Fatalf("export row=%+v", row)
	}
}
