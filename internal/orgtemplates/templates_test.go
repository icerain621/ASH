package orgtemplates

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

func TestCatalogHasThreeTemplates(t *testing.T) {
	cat := Catalog()
	if len(cat) != 3 {
		t.Fatalf("catalog=%d want 3", len(cat))
	}
	for _, id := range []string{IDSmallTeam, IDMidEnterprise, IDStrongCompliance} {
		if _, ok := Get(id); !ok {
			t.Fatalf("missing template %s", id)
		}
	}
}

func TestProvisionSmallTeam(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	var result ProvisionResult
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = Provision(tx, IDSmallTeam, ProvisionRequest{
			OrgName: "Demo Startup", OrgSlug: "demo-startup", ActorID: "dev-user", SpaceID: "local",
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Org.Slug != "demo-startup" || result.TemplateID != IDSmallTeam {
		t.Fatalf("result=%+v", result)
	}
	if len(result.Spaces) != 1 {
		t.Fatalf("spaces=%d want 1", len(result.Spaces))
	}
	if len(result.Roles) < 3 { // admin + operator + reviewer
		t.Fatalf("roles=%d want >=3", len(result.Roles))
	}
	var scopes int64
	if err := db.Model(&store.ResourceScope{}).Where("space_id = ?", result.Spaces[0].ID).Count(&scopes).Error; err != nil {
		t.Fatal(err)
	}
	if scopes < 4 {
		t.Fatalf("scopes=%d want >=4 (space + scenarios)", scopes)
	}
	var audits int64
	if err := db.Model(&store.AuditLog{}).Where("event_type = ?", "org.template_provisioned").Count(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if audits != 1 {
		t.Fatalf("audits=%d want 1", audits)
	}
}

func TestProvisionUnknownRejected(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := Provision(tx, "nope", ProvisionRequest{})
		return err
	})
	if err == nil {
		t.Fatal("expected unknown template error")
	}
}
