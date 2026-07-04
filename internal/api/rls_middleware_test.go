package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestLookupSpaceOrgID(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(db, loader)
	now := time.Now().UTC()
	org := store.Org{ID: "org_lookup", Name: "Lookup Org", Slug: "lookup", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_lookup", OrgID: org.ID, Name: "Lookup Space", Slug: "lookup-space", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&org).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	orgID, err := h.lookupSpaceOrgID(c, space.ID)
	if err != nil {
		t.Fatal(err)
	}
	if orgID != org.ID {
		t.Fatalf("orgID=%q want %q", orgID, org.ID)
	}
}
